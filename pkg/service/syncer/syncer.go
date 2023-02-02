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
	"github.com/bsn-si/IPEHR-stat/pkg/localDB"
	"github.com/pkg/errors"
)

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
	db           *localDB.DB
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

func New(db *localDB.DB, ethClient *ethclient.Client, cfg Config) *Syncer {
	s := Syncer{
		db:        db,
		ethClient: ethClient,
		addrList:  sync.Map{},
		blockNum:  big.NewInt(int64(cfg.StartBlock)),
	}

	lastBlock, err := db.SyncLastBlockGet()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = db.SyncLastBlockSet(cfg.StartBlock)
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
	var bigInt1 = big.NewInt(1)

	log.Printf("[SYNC] Starting sync from block number: %d", s.blockNum)

	go func() {
		for {
			// get the full block details, using a custom jsonrpc ID as a test
			block, err := s.ethClient.BlockByNumber(ctx, s.blockNum)
			if err != nil {
				if err.Error() == "not found" {
					time.Sleep(BlockNotFoundTimeout)
					continue
				} else {
					log.Printf("[SYNC] Block %d %v get error:", s.blockNum, err)
					log.Printf("[SYNC] BlockByNumber error: %v Sleeping %s...", err, BlockGetErrorTimeout)
					time.Sleep(BlockGetErrorTimeout)
					continue
				}
			}

			ts := time.Unix(int64(block.Time()), 0)

			for _, blockTx := range block.Transactions() {
				if blockTx.To() == nil {
					// contract creation
					continue
				}

				contractName, ok := s.addrList.Load(*blockTx.To())
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

				decodedSig := blockTx.Data()[:4]
				decodedData := blockTx.Data()[4:]

				var _abi *abi.ABI

				switch contractName {
				case "ehrIndex":
					_abi = s.ehrABI
				case "users":
					_abi = s.usersABI
				case "dataStore":
					_abi = s.dataStoreABI
				}

				method, err := _abi.MethodById(decodedSig)
				if err != nil {
					log.Println("abi.MethodById error: ", err)
					continue
				}

				switch method.Name {
				case "multicall":
					err = s.procMulticall(_abi, method, decodedData, ts)
					if err != nil {
						log.Fatal("[SYNC] procMulticall error: ", err)
					}
				case "addEhrDoc":
					err = s.procAddEhrDoc(method, decodedData, ts)
					if err != nil {
						log.Fatal("[SYNC] procAddEhrDoc error: ", err)
					}
				case "userNew":
					err = s.procUserNew(method, decodedData, ts)
					if err != nil {
						log.Fatal("[SYNC] procUserNew error: ", err)
					}
				case "dataUpdate":
					err = s.procDataUpdate(method, decodedData)
					if err != nil {
						log.Fatal("[SYNC] procDataUpdate error: ", err)
					}
				}
			}

			log.Printf("[SYNC] new block %v %v txs %d", block.Number().Int64(), time.Unix(int64(block.Time()), 0).Format("2006-01-02 15:04:05"), len(block.Transactions()))

			err = s.db.SyncLastBlockSet(s.blockNum.Uint64())
			if err != nil {
				log.Fatal("[SYNC] SyncLastBlockSet error: ", err)
			}

			s.blockNum.Add(s.blockNum, bigInt1)
		}
	}()
}

func (s *Syncer) procMulticall(_abi *abi.ABI, method *abi.Method, inputData []byte, ts time.Time) error {
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
			err = s.procAddEhrDoc(method, decodedData, ts)
			if err != nil {
				return fmt.Errorf("procAddEhrDoc error: %w", err)
			}
		case "userNew":
			err = s.procUserNew(method, decodedData, ts)
			if err != nil {
				return fmt.Errorf("procUserNew error: %w", err)
			}
		}
	}

	return nil
}

func (s *Syncer) procAddEhrDoc(method *abi.Method, inputData []byte, ts time.Time) error { //nolint
	log.Println("[STAT] new EHR document registered")

	err := s.db.StatDocumentsCountIncrement(ts)
	if err != nil {
		return fmt.Errorf("StatDocumentsCountIncrement error: %w", err)
	}

	return nil
}

func (s *Syncer) procUserNew(method *abi.Method, inputData []byte, ts time.Time) error {
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
		err := s.db.StatPatientsCountIncrement(ts)
		if err != nil {
			return fmt.Errorf("StatPatientsCountIncrement error: %w", err)
		}
	}

	return nil
}

func (s *Syncer) procDataUpdate(method *abi.Method, inputData []byte) error {
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
	switch v := args[0].(type) {
	case [32]byte:
	default:
		return errors.Errorf("unexpected type %T of arg[0] bytes32 groupID", v)
	}

	// dataID
	switch v := args[1].(type) {
	case [32]byte:
	default:
		return errors.Errorf("unexpected type %T of arg[1] bytes32 dataID", v)
	}

	// ehrID
	var ehrID string

	switch v := args[2].(type) {
	case [32]byte:
		u, err := uuid.FromBytes(v[:16])
		if err != nil {
			return fmt.Errorf("ehrID uuid.FromBytes error: %w ehrID: %x", err, v[:16])
		}

		ehrID = u.String()
	default:
		return errors.Errorf("unexpected type %T of arg[2] bytes32 ehrID", v)
	}

	// data
	var compressedData []byte

	switch v := args[3].(type) {
	case []byte:
		compressedData = v
	default:
		return errors.Errorf("unexpected type %T of arg[3] bytes data", v)
	}

	data, err := decompress(compressedData)
	if err != nil {
		return fmt.Errorf("data decompression error: %w", err)
	}

	var nodeObj treeindex.ObjectNode

	err = msgpack.Unmarshal(data, &nodeObj)
	if err != nil {
		return fmt.Errorf("data unmarshal error: %w", err)
	}

	switch nodeObj.GetNodeType() {
	case treeindex.EHRNodeType:
		var ehrNode treeindex.EHRNode

		err = msgpack.Unmarshal(data, &ehrNode)
		if err != nil {
			return fmt.Errorf("ehrNode unmarshal error: %w", err)
		}

		err = treeindex.DefaultEHRIndex.AddEHRNode(&ehrNode)
		if err != nil {
			return fmt.Errorf("AddEHRNode error: %w", err)
		}
	case treeindex.CompostionNodeType:
		var cmpNode treeindex.CompositionNode

		err = msgpack.Unmarshal(data, &cmpNode)
		if err != nil {
			return fmt.Errorf("cmpNode unmarshal error: %w", err)
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
