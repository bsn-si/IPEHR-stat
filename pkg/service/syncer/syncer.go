package syncer

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"io"

	"fmt"
	"log"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/bsn-si/IPEHR-gateway/src/pkg/storage/treeindex"
	"github.com/bsn-si/IPEHR-stat/internal/models"
	"github.com/pkg/errors"
)

type SyncerRepo interface { //nolint:revive
	SyncLastBlockGet(ctx context.Context) (uint64, error)
	SyncLastBlockSet(ctx context.Context, lastSyncedBlock uint64) error

	StatPatientsCountIncrement(ctx context.Context, timestamp time.Time) error
	StatDocumentsCountIncrement(ctx context.Context, timestamp time.Time) error
}

type TreeIndexChunkRepositpry interface {
	AddNewIndexObject(ctx context.Context, chunk models.IndexChunk) error
	GetAllIndexObjects(ctx context.Context) ([]models.IndexChunk, error)
}

type Config struct {
	Endpoint   string
	StartBlock uint64
	Contracts  []struct {
		Name    string
		Address string
		AbiPath string
	}
}

type Syncer struct {
	repo         SyncerRepo
	chunkRepo    TreeIndexChunkRepositpry
	ethClient    *ethclient.Client
	addrList     sync.Map
	ehrABI       *abi.ABI
	usersABI     *abi.ABI
	dataStoreABI *abi.ABI
	blockNum     *big.Int
}

const (
	BlockNotFoundTimeout = time.Second * 15
	BlockGetErrorTimeout = time.Second * 30

	RolePatient uint8 = 0
	RoleDoctor  uint8 = 1
)

func New(repo SyncerRepo, chunkRepo TreeIndexChunkRepositpry, ethClient *ethclient.Client, cfg Config) *Syncer {
	s := Syncer{
		repo:      repo,
		chunkRepo: chunkRepo,
		ethClient: ethClient,
		addrList:  sync.Map{},
		blockNum:  big.NewInt(int64(cfg.StartBlock)),
	}

	lastBlock, err := repo.SyncLastBlockGet(context.Background())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = repo.SyncLastBlockSet(context.Background(), cfg.StartBlock)
			if err != nil {
				log.Fatal("[SYNC] SyncLastBlockSet error: ", err)
			}
		} else {
			log.Fatal("SyncLastBlockGet error: ", err)
		}
	}

	if lastBlock > s.blockNum.Uint64() {
		s.blockNum = big.NewInt(int64(lastBlock))
	}

	for _, c := range cfg.Contracts {
		s.addrList.Store(common.HexToAddress(c.Address), c.Name)

		abiJSON, err := os.ReadFile(c.AbiPath)
		if err != nil {
			log.Fatalf("abiSON read file '%s' error: %v", c.AbiPath, err)
		}

		abi, err := abi.JSON(bytes.NewReader(abiJSON))
		if err != nil {
			log.Fatal("abi.JSON error: ", err)
		}

		switch c.Name {
		case "ehrIndex":
			s.ehrABI = &abi
		case "users":
			s.usersABI = &abi
		case "dataStore":
			s.dataStoreABI = &abi
		}
	}

	switch {
	case s.ehrABI == nil:
		log.Fatal("Error: contract 'ehrIndex' definition is not found in config")
	case s.usersABI == nil:
		log.Fatal("Error: contract 'users' definition is not found in config")
	case s.dataStoreABI == nil:
		log.Fatal("Error: contract 'dataStore' definition is not found in config")
	}

	return &s
}

func (s *Syncer) Start(ctx context.Context) {
	log.Printf("[SYNC] Load tree index state from storage")

	if err := s.loadIndexDataFromStorage(ctx); err != nil {
		log.Fatal(err)
	}

	log.Printf("[SYNC] Starting sync from block number: %d", s.blockNum)

	go func() {
		bInt := big.NewInt(1)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				s.tryProccessNextBlock(ctx, bInt)
			}
		}
	}()
}

func (s *Syncer) loadIndexDataFromStorage(ctx context.Context) error {
	chucks, err := s.chunkRepo.GetAllIndexObjects(ctx)
	if err != nil {
		return fmt.Errorf("cannot load index data from storage: %w", err)
	}

	for _, chunk := range chucks {
		if !chunk.Validate() {
			return fmt.Errorf("data chunk invalid: %v", chunk.Key) //nolint
		}

		if err := s.unmarshalDataAndStoreInIndex(chunk.EhrID, chunk.Data); err != nil {
			return fmt.Errorf("cannot store chunk into index: %w", err)
		}
	}

	return nil
}

func (s *Syncer) tryProccessNextBlock(ctx context.Context, bInt *big.Int) {
	// get the full block details, using a custom jsonrpc ID as a test
	block, err := s.ethClient.BlockByNumber(ctx, s.blockNum)
	if err != nil {
		if err.Error() == "not found" {
			time.Sleep(BlockNotFoundTimeout)
		} else {
			log.Printf("[SYNC] Block %d %v get error:", s.blockNum, err)
			log.Printf("[SYNC] BlockByNumber error: %v Sleeping %s...", err, BlockGetErrorTimeout)
			time.Sleep(BlockGetErrorTimeout)
		}

		return
	}

	ts := time.Unix(int64(block.Time()), 0)

	for _, blockTx := range block.Transactions() {
		if blockTx.To() == nil {
			// contract creation
			continue
		}

		contractNameRow, ok := s.addrList.Load(*blockTx.To())
		if !ok {
			continue
		}

		contractName, ok := contractNameRow.(string)
		if !ok {
			continue
		}

		receipt, err := s.ethClient.TransactionReceipt(ctx, blockTx.Hash())
		if err != nil {
			log.Printf("[SYNC] tx %s receipt get error: %v", blockTx.Hash().String(), err)
		}

		if receipt.Status == types.ReceiptStatusFailed {
			continue
		}

		s.processTransactionBlock(ctx, contractName, blockTx, ts)
	}

	log.Printf("[SYNC] new block %v %v txs %d", block.Number().Int64(), time.Unix(int64(block.Time()), 0).Format("2006-01-02 15:04:05"), len(block.Transactions()))

	if err := s.repo.SyncLastBlockSet(ctx, s.blockNum.Uint64()); err != nil {
		log.Fatal("[SYNC] SyncLastBlockSet error: ", err)
	}

	s.blockNum.Add(s.blockNum, bInt)
}

func (s *Syncer) processTransactionBlock(ctx context.Context, contractName string, blockTx *types.Transaction, ts time.Time) {
	decodedSig := blockTx.Data()[:4]
	decodedData := blockTx.Data()[4:]

	abi := s.getAbiByForContract(contractName)

	method, err := abi.MethodById(decodedSig)
	if err != nil {
		log.Println("abi.MethodById error: ", err)
		return
	}

	switch method.Name {
	case "multicall":
		err = s.procMulticall(ctx, abi, method, decodedData, ts)
		if err != nil {
			log.Fatal("[SYNC] procMulticall error: ", err)
		}
	case "addEhrDoc":
		err = s.procAddEhrDoc(ctx, method, decodedData, ts)
		if err != nil {
			log.Fatal("[SYNC] procAddEhrDoc error: ", err)
		}
	case "userNew":
		err = s.procUserNew(ctx, method, decodedData, ts)
		if err != nil {
			log.Fatal("[SYNC] procUserNew error: ", err)
		}
	case "dataUpdate":
		err = s.procDataUpdate(ctx, method, decodedData)
		if err != nil {
			log.Fatal("[SYNC] procDataUpdate error: ", err)
		}
	}
}

func (s *Syncer) getAbiByForContract(contractName string) *abi.ABI {
	switch contractName {
	case "ehrIndex":
		return s.ehrABI
	case "users":
		return s.usersABI
	case "dataStore":
		return s.dataStoreABI
	}

	return nil
}

func (s *Syncer) procMulticall(ctx context.Context, _abi *abi.ABI, method *abi.Method, inputData []byte, ts time.Time) error {
	args, err := method.Inputs.Unpack(inputData)
	if err != nil {
		return fmt.Errorf("UnpackValues error: %w", err)
	}

	for _, m := range args[0].([][]byte) {
		decodedSig := m[:4]
		decodedData := m[4:]

		method, err = _abi.MethodById(decodedSig)
		if err != nil {
			return fmt.Errorf("abi.MethodById error: %w", err)
		}

		switch method.Name {
		case "addEhrDoc":
			err = s.procAddEhrDoc(ctx, method, decodedData, ts)
			if err != nil {
				return fmt.Errorf("procAddEhrDoc error: %w", err)
			}
		case "userNew":
			err = s.procUserNew(ctx, method, decodedData, ts)
			if err != nil {
				return fmt.Errorf("procUserNew error: %w", err)
			}
		}
	}

	return nil
}

func (s *Syncer) procAddEhrDoc(ctx context.Context, method *abi.Method, inputData []byte, ts time.Time) error { //nolint
	log.Println("[STAT] new EHR document registered")

	err := s.repo.StatDocumentsCountIncrement(ctx, ts)
	if err != nil {
		return fmt.Errorf("StatDocumentsCountIncrement error: %w", err)
	}

	return nil
}

func (s *Syncer) procUserNew(ctx context.Context, method *abi.Method, inputData []byte, ts time.Time) error {
	log.Println("[STAT] new patient registered")

	args, err := method.Inputs.Unpack(inputData)
	if err != nil {
		return fmt.Errorf("UnpackValues error: %w", err)
	}

	// interface: function userNew(address addr, bytes32 IDHash, Role role, Attributes.Attribute[] calldata attrs, address signer, bytes calldata signature)
	if len(args) < 3 {
		return fmt.Errorf("args length(%d) < 3", len(args)) //nolint
	}

	role := args[2].(uint8)

	if role == RolePatient {
		err := s.repo.StatPatientsCountIncrement(ctx, ts)
		if err != nil {
			return fmt.Errorf("StatPatientsCountIncrement error: %w", err)
		}
	}

	return nil
}

func (s *Syncer) procDataUpdate(ctx context.Context, method *abi.Method, inputData []byte) error {
	log.Println("[STAT] dataIndex update")

	args, err := method.Inputs.Unpack(inputData)
	if err != nil {
		return fmt.Errorf("UnpackValues error: %w", err)
	}

	// interface: function unction dataUpdate(bytes32 groupID, bytes32 dataID, bytes32 ehrID, bytes data, address signer, bytes calldata signature)
	if len(args) != 6 {
		return fmt.Errorf("args length(%d) != 6", len(args)) //nolint
	}

	// groupID
	groupID, err := tryGetUUIDStr(args[0])
	if err != nil {
		return errors.Wrap(err, "cannot get groupID")
	}

	// dataID
	dataID, err := tryGetUUIDStr(args[1])
	if err != nil {
		return errors.Wrap(err, "cannot get dataID")
	}

	// ehrID
	ehrID, err := tryGetUUIDStr(args[2])
	if err != nil {
		return errors.Wrap(err, "cannot get ehr_id")
	}

	// data
	data, err := getMessageData(args[3])
	if err != nil {
		return errors.Wrap(err, "cannot get message data")
	}

	idxChunk := models.NewIndexChunk(groupID, dataID, ehrID, data)
	if err := s.chunkRepo.AddNewIndexObject(ctx, idxChunk); err != nil {
		return errors.Wrap(err, "cannot save index chunk into sotrage")
	}
  
  return s.unmarshalDataAndStoreInIndex(ehrID, data)
}

func (s *Syncer) unmarshalDataAndStoreInIndex(ehrID string, data []byte) error {
	var nodeObj treeindex.ObjectNode

  err = msgpack.Unmarshal(data, &nodeObj)
	if err != nil {
		return fmt.Errorf("msgpack.Unmarshal error: %w", err)
	}

	switch nodeObj.GetNodeType() {
	case treeindex.NodeTypeEHR:
		var ehrNode treeindex.EHRNode

		err = msgpack.Unmarshal(data, &ehrNode)
		if err != nil {
			return fmt.Errorf("ehrNode Unmarshal error: %w", err)
		}

		if err := treeindex.DefaultEHRIndex.AddEHRNode(&ehrNode); err != nil {
			return fmt.Errorf("AddEHRNode error: %w", err)
		}
	case treeindex.NodeTypeCompostion:
		var cmpNode treeindex.CompositionNode

		err = msgpack.Unmarshal(data, &cmpNode)
		if err != nil {
			return fmt.Errorf("cmpNode Unmarshal error: %w", err)
		}

		ehrNodes, err := treeindex.DefaultEHRIndex.GetEHRs(ehrID)
		if err != nil {
			return fmt.Errorf("treeindex GetEHRs error: %w ehrID: %s", err, ehrID)
		}

		if len(ehrNodes) != 1 {
			return errors.Errorf("ehrNode with ehrID %s not nound", ehrID)
		}

		err = ehrNodes[0].AddCompositionNode(&cmpNode)
		if err != nil {
			return fmt.Errorf("AddCompositionNode error: %w", err)
		}
	default:
		return errors.Errorf("unsupported node type: %v", nodeObj.GetNodeType())
	}

	return nil
}

func tryGetUUIDStr(data interface{}) (string, error) {
	v, ok := data.([32]byte)
	if !ok {
		return "", errors.Errorf("unexpected type %T of data bytes32 uuid", data)
	}

	ehrID, err := uuid.FromBytes(v[:16])
	if err != nil {
		return "", fmt.Errorf("uuid.FromBytes error: %w value: %x", err, v[:16])
	}

	return ehrID.String(), nil
}

func getMessageData(val interface{}) ([]byte, error) {
	compressedData, ok := val.([]byte)

	if !ok {
		return nil, errors.Errorf("unexpected type %T of val bytes data", val)
	}

	data, err := decompress(compressedData)
	if err != nil {
		return nil, fmt.Errorf("data decompression error: %w", err)
	}

	return data, nil
}

func decompress(data []byte) ([]byte, error) {
	buf := bytes.NewReader(data)

	zr, err := gzip.NewReader(buf)
	if err != nil {
		return nil, fmt.Errorf("gzip.NewReader error: %w", err)
	}
	defer zr.Close()

	decompressed, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("io.ReadAll error: %w", err)
	}

	return decompressed, nil
}
