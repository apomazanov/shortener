package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func main() {
	endpoint := "http://localhost:8080/"
	// контейнер данных для запроса
	data := url.Values{}
	// приглашение в консоли
	fmt.Println("Введите длинный URL")
	// открываем потоковое чтение из консоли
	reader := bufio.NewScanner(os.Stdin)
	// читаем строку из консоли
	var long string
	if reader.Scan() {
		long = reader.Text()
	}
	if err := reader.Err(); err != nil {
		fmt.Println(err)
		return 
	}
	// заполняем контейнер данными
	data.Set("url", long)
	// добавляем HTTP-клиент
	client := &http.Client{Timeout: 5 * time.Second}
	// пишем запрос
	// запрос методом POST должен, помимо заголовков, содержать тело
	// тело должно быть источником потокового чтения io.Reader
	request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		fmt.Println(err)
		return 
	}
	// в заголовках запроса указываем кодировку
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	// отправляем запрос и получаем ответ
	response, err := client.Do(request)
	if err != nil {
		fmt.Println(err)
		return 
	}
	defer func() {
		io.Copy(io.Discard, response.Body)
		response.Body.Close()
	}()
	// выводим код ответа
	fmt.Println("Статус-код ", response.Status)
	// читаем поток из тела ответа
	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err)
		return 
	}
	// и печатаем его
	fmt.Println(string(body))
}
