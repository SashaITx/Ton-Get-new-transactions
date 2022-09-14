package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"log"
	"time"
)

func main() {

	client := liteclient.NewConnectionPool()

	configUrl := "https://ton-blockchain.github.io/global.config.json"

	err := client.AddConnectionsFromConfigUrl(context.Background(), configUrl)
	if err != nil {
		log.Fatalln("connection err: ", err.Error())
		return
	}

	api := ton.NewAPIClient(client)

	var accAddress string
	fmt.Println("Please enter account address ...")
	fmt.Scan(&accAddress)
	addr := address.MustParseAddr(accAddress) // MustParseAddr уже проверяет len адреса.

	listTx := make([]*tlb.Transaction, 2, 2)

	for i := 0; ; i++ {
		fmt.Printf("Request No %v", i)

		// Нужно спарсить текущее состояние матерчейна для использования гет методов применительно к конкретному адресу.
		master, err := api.CurrentMasterchainInfo(context.Background())
		if err != nil {
			log.Fatalln("get block err:", err.Error())
			return
		}

		account, err := api.GetAccount(context.Background(), master, addr)

		switch {
		case !account.IsActive || !account.State.IsValid:
			fmt.Println("Account is not active.")
			return
		case i == 0:
			fmt.Println("Getting list of transactions...")
		}

		lastTransaction, err := api.ListTransactions(context.Background(), addr, 1, account.LastTxLT, account.LastTxHash)
		if err != nil {
			log.Printf("send err: %s", err.Error())
			return
		}

		// На первой итерации мы помещаем последнюю транзакцию аккаунта на нулевой индекс в слайсе.
		if i == 0 {
			fmt.Printf("Last transaction information: %s\n", lastTransaction[0].String())
			listTx[0] = lastTransaction[0]
			fmt.Println("Waiting for a new transaction...")
			continue
			// На последующих итерациях мы помещаем последнюю транзакцию аккаунта на первый индекс в слайсе.
		} else {
			listTx[1] = lastTransaction[0]
		}

		// Далее сравним две последние транзакции аккаунта. Если одинаковые значит новых транзакций нет - ждем и продолжаем запрашивать информацию.
		if bytes.Equal(listTx[0].Hash, listTx[1].Hash) {
			time.Sleep(3 * time.Second)
			// Если новая транзакция отличается от предыдущей выводим её в консоль, записываем новую транзакцию в 0-ой индекс, ждем и продолжаем запрашивать.
		} else {
			fmt.Printf("Fined new transaction: %s\n", lastTransaction[0].String())
			listTx[0] = listTx[1]
			time.Sleep(3 * time.Second)
			fmt.Println("Waiting for a new transaction...")
		}
	}
}
