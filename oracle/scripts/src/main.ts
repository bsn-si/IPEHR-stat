import { readFileSync } from "fs"
import * as ethers from "ethers"
import * as path from "path"

const KEY_JSON = readFileSync(path.join(__dirname, "Account.json"), "utf-8")
const KEY_PASS = "qwerty12345"

async function main() {
  const wallet = await ethers.Wallet.fromEncryptedJson(KEY_JSON, KEY_PASS)
  const provider = new ethers.providers.WebSocketProvider("ws://127.0.0.1:8546")
  const signer = wallet.connect(provider)

  const gasPrice = await provider.getGasPrice()
  // const balance = await signer.getBalance();
  const balance = await provider.getBalance("0x887dd6d1C11508e7021C9A8ccd88Fa3bE3bAFDD6")

  console.log({
    gasPrice: ethers.utils.formatEther(gasPrice),
    balance: ethers.utils.formatEther(balance),
  })

  const transaction = await signer.sendTransaction({
    // gasLimit: ethers.utils.parseUnits("1", "gwei"),
    // gasPrice,

    to: "0x887dd6d1C11508e7021C9A8ccd88Fa3bE3bAFDD6",
    value: ethers.utils.parseUnits("25", "ether"),
  })

  const receipt = await transaction.wait(1)
  console.log(receipt)
}

main()
  .catch(error => {
    console.error(error)
    process.exit(1)
  })
  .then(() => {
    process.exit()
  })
