package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Параметры тестирования
const secCount = 5      // Продолжительность тестирования в  секундах
const reqCount = 200    // Количество запросов в секунду
const isBalance = false // Использовать "балансировшик нагрузки": false - все запросы на первый порт из списка, true - все запросы равномерно делятся по всем портам
const dbReqMode = "v1"  // Порядок запросов к БД : "v1" - параллельный, "v2" -последовательный

// var getUrl = GetUrl(isBalance, "8000", "8001", "8002", "8003", "8004") // Создание "балансировщика нагрузки"
var getUrl = GetUrl(isBalance, "8010", "8011", "8012", "8013", "8014") // Создание "балансировщика нагрузки"
var stat [secCount]map[string]int                                      // Объявление переменной для формирования статистики выполнения запросов. 201 - успех, 500 - ошибка БД, error - ошибка API сервера

// Запрос на сервер и фиксация статуса его выполнения
func TestRequest(reqUrl string, sec int, mutexChangeStat *sync.Mutex) {
	reqBody, err := json.Marshal(map[string]any{
		"device_id": uuid.New(),
		"value":     rand.Intn(10000),
	})
	if err != nil {
		fmt.Printf("Test request error: %s\n", err)
		return
	}

	resp, err := http.Post(reqUrl, "application/json", bytes.NewBuffer(reqBody))
	statusCode := "error"
	if err == nil {
		resp.Body.Close()
		statusCode = resp.Status
	}
	mutexChangeStat.Lock()
	stat[sec][statusCode]++
	mutexChangeStat.Unlock()
}

// Имитация балансировки нагрузки по методу Round Robin без контроля доступности сервера
func GetUrl(isBalance bool, ports ...string) func() string {
	portIndex := 0
	return func() string {
		if isBalance {
			portIndex++
			if portIndex > len(ports)-1 {
				portIndex = 0
			}
		}
		return fmt.Sprintf("http://localhost:%s/%s/values", ports[portIndex], dbReqMode)
	}
}
func main() {
	fmt.Printf("%s Begin test(Second count: %d. Requests per second: %d. Balance: %t. DB request mode '%s' )\n", time.Now().Format("03:04:05.999"), secCount, reqCount, isBalance, dbReqMode)

	var mutexChangeStat sync.Mutex
	wg := sync.WaitGroup{}

	// Цикл секунд
	for sec := 0; sec < secCount; sec++ {
		start := time.Now()
		stat[sec] = make(map[string]int, 3)
		fmt.Printf("%s Second %d\n", time.Now().Format("03:04:05.999"), sec+1)
		// Цикл запросов
		for req := 0; req < reqCount; req++ {
			wg.Add(1)
			go func() {
				reqUrl := getUrl()
				TestRequest(reqUrl, sec, &mutexChangeStat)
				wg.Done()
			}()
		}
		time.Sleep(time.Second - time.Since(start))
	}
	// Ожидание завершения работы горутин
	fmt.Printf("%s Wait goroutines\n", time.Now().Format("03:04:05.999"))
	wg.Wait()
	fmt.Printf("%s End test\n", time.Now().Format("03:04:05.999"))
	//Вывод результатов
	fmt.Printf("Statistics\n")
	total := make(map[string]int, 3)
	for sec := range stat {
		fmt.Printf("Second %d\n", sec+1)
		for key, value := range stat[sec] {
			total[key] = total[key] + value
			fmt.Printf("	%s : %v\n", key, value)
		}
	}

	fmt.Printf("Total\n")
	for key, value := range total {
		fmt.Printf("	%s : %v\n", key, value)

	}

}
