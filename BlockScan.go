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
	// Устанавливаем соединение с блокчейном. Это необходимо для получения данных о состоянии блокчейна от ноды.
	client := liteclient.NewConnectionPool()

	configUrl := "https://ton-blockchain.github.io/global.config.json"

	// Под капотом мы парсим кофиги -> пробуем подключится к лайтсерверам ТОНа в разных рутинах до установления подключения
	err := client.AddConnectionsFromConfigUrl(context.Background(), configUrl)
	if err != nil {
		log.Fatalln("connection err: ", err.Error())
		return
	}
	/*
		???
		Сложно понять что тут под капотом. Нужно уточнить.
		Судя по всему это обертка над лайтклиентом с добавлением 3х полей о текущем состоянии маcтерчейна.
		Лайтклиент (интерфейс с одним методом Do) - протокол общения (request to liteserver) с нодой, предусматривающий проверку рутхэша.
		Метод Do возвращает LiteResponse - структура с id и dat'ой.
		???
	*/
	api := ton.NewAPIClient(client)

	// Нужно спарсить текущее состояние матерчейна для использования гет методов применительно к конкретному адресу.
	master, err := api.GetMasterchainInfo(context.Background())
	if err != nil {
		log.Fatalln("get block err:", err.Error())
		return
	}

	var accAddress string
	fmt.Println("Please enter account address ...")
	fmt.Scan(&accAddress)
	addr := address.MustParseAddr(accAddress) // MustParseAddr уже проверяет len адреса.
	account, err := api.GetAccount(context.Background(), master, addr)

	switch {
	case !account.IsActive || !account.State.IsValid:
		fmt.Println("Account is not active.")
		return
	default:
		fmt.Println("Getting list of transactions...")
	}

	var shards []*tlb.BlockInfo
	listTx := make([]*tlb.Transaction, 2, 2)

	for i := 0; ; i++ {
		master, err := api.GetMasterchainInfo(context.Background())
		if err != nil {
			log.Fatalln("get block err:", err.Error())
			return
		}

		shards, err = api.GetBlockShardsInfo(context.Background(), master)
		if err != nil {
			log.Fatalln("get shards err:", err.Error())
			return
		}

		if len(shards) == 0 {
			log.Println("master block without shards, waiting for next...")
			time.Sleep(3 * time.Second)
			continue
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