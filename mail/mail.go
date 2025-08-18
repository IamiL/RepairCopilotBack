package main

import (
	"log"
	"net/smtp"
)

func main() {
	// Конфигурация SMTP для Gmail
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	auth := smtp.PlainAuth("", "ivan2011avatar@gmail.com", "tsep nuqs bmvy dcbr", smtpHost)

	// Адрес отправителя и получателя
	from := "ivan2011avatar@gmail.com"
	to := "iamil50113@gmail.com"

	// Тело письма
	message := []byte("To: " + to + "\r\n" +
		"Subject: Тестовое письмо через Gmail\r\n" +
		"\r\n" +
		"Привет! Это тестовое письмо, отправленное через Gmail SMTP.\r\n")

	// Отправка письма через Gmail
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, message)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Письмо успешно отправлено через Gmail!")
}
