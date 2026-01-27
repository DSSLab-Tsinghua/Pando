/*
generating threshold prf keys
*/

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"pando/src/cryptolib"
	"pando/src/utils"
)

const (
	helpText = `
Generating keys for ecdsa
ecdsagen [initID] [endID]
`
)

func main() {

	helpPtr := flag.Bool("help", false, helpText)
	flag.Parse()

	if *helpPtr || len(os.Args) < 2 {
		log.Printf(helpText)
		return
	}

	initid := "0"
	if len(os.Args) > 1 {
		initid = os.Args[1]
	}

	endid := initid
	if len(os.Args) > 2 {
		endid = os.Args[2]
	}

	cryptolib.SetHomeDir()

	if initid == "new" {
		cryptolib.GenerateKey(-1)
		return
	}
	bid, _ := utils.StringToInt64(initid)
	eid, _ := utils.StringToInt64(endid)

	for i := bid; i <= eid; i++ {
		cryptolib.GenerateKey(i)
	}

	var sig []byte
	var result bool

	message := utils.StringToBytes("test message")
	for j := bid; j <= eid; j++ {
		cryptolib.StartECDSA(j)

		sig = cryptolib.GenSig(j, message)
		fmt.Println("[Generate] Generating sigs using keys %d....", j)

		result = cryptolib.VerifySig(j, message, sig)
		if result {
			fmt.Println("[Verify] Verify newly generated sig success...", j)
		}
	}
}
