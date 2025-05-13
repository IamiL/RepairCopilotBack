package httpHandler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"repairCopilotBot/internal/JWTsecret"
	http_api "repairCopilotBot/internal/pkg/http-api"
	jwtToken "repairCopilotBot/internal/pkg/jwt"
	"repairCopilotBot/internal/pkg/logger/sl"
	"strconv"
	"sync"
	"time"
)

type Message struct {
	Body  string    `json:"body"`
	Time  time.Time `json:"time"`
	IsBot bool      `json:"isBot"`
}

type MessagesStorage struct {
	Storage map[string][]Message
	Mu      sync.RWMutex
}

type ValidationError struct {
	Detail []struct {
		Loc  []string `json:"loc"`
		Msg  string   `json:"msg"`
		Type string   `json:"type"`
	} `json:"detail"`
}

func (e ValidationError) Error() string {
	var msgs []string
	for _, detail := range e.Detail {
		msgs = append(msgs, fmt.Sprintf("%s: %s (location: %v)", detail.Type, detail.Msg, detail.Loc))
	}
	return fmt.Sprintf("Validation error: %v", msgs)
}

type StartChatResponse struct {
	Resp string `json:"resp"`
}

func StartChatHandler(
	log *slog.Logger,
	messages *MessagesStorage,
	JWTSecret *JWTsecret.JWTSecret,
) func(
	w http.ResponseWriter, r *http.Request,
) {
	return func(w http.ResponseWriter, r *http.Request) {
		//var req EditBuildingRequest
		//if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		//	log.Info("Error decoding body: ", sl.Err(err))
		//	http_api.HandleError(
		//		w,
		//		http.StatusBadRequest,
		//		"Invalid request body",
		//	)
		//	return
		//}

		rand.Seed(time.Now().UnixNano())
		var chatID string
		for i := 0; i < 10; i++ {
			chatID += strconv.Itoa(rand.Intn(10)) // Генерация случайной цифры от 0 до 9
		}

		log.Info(`Сгенерирован id чата - `, chatID)
		//num, err := strconv.Atoi(chatID) // Преобразование строки в целое число
		//if err != nil {
		//	fmt.Println("Ошибка:", err)
		//	return
		//}

		restCh := make(chan string)

		go func(ch chan string) {
			//data := url.Values{}
			//data.Set("user_id", chatID)
			//
			//body := bytes.NewBufferString(data.Encode())
			//
			//req, err := http.NewRequest("POST", "http://localhost:8000/start_dialog", body)
			//if err != nil {
			//	fmt.Printf("Ошибка при создании запроса: %v\n", err)
			//	return
			//}
			//
			//client := &http.Client{}
			//resp, err := client.Do(req)
			//if err != nil {
			//	fmt.Printf("Ошибка при выполнении запроса: %v\n", err)
			//	return
			//}
			//defer resp.Body.Close()

			baseURL := "http://localhost:8000/start_dialog"
			params := url.Values{}
			params.Add("user_id", chatID)

			resp, err := http.Post(fmt.Sprintf("%s?%s", baseURL, params.Encode()), "application/json", nil)
			if err != nil {
				fmt.Println(fmt.Errorf("request failed: %v", err))

				ch <- "Сервер недоступен."

				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(fmt.Errorf("failed to read response: %v", err))
			}
			if resp.StatusCode == http.StatusUnprocessableEntity { // 422
				var validationErr ValidationError
				if err := json.Unmarshal(body, &validationErr); err != nil {
					fmt.Println(fmt.Errorf("failed to parse validation error: %v", err))
				}
				fmt.Println(validationErr)
			}

			if resp.StatusCode == http.StatusOK {
				//respMsgBody, err := io.ReadAll(resp.Body)
				//if err != nil {
				//	fmt.Println(fmt.Errorf("failed to read response: %v", err))
				//}

				//buf := make([]byte, 100000) // Размер буфера
				//var body []byte
				//for {
				//	n, err := resp.Body.Read(buf)
				//	if err == io.EOF {
				//		break // Конец ответа
				//	}
				//	if err != nil {
				//		fmt.Printf("Ошибка чтения: %v\n", err)
				//		return
				//	}
				//	body = append(body, buf[:n]...) // Добавляем прочитанные данные в body
				//}
				fmt.Printf("Тело ответа:\n%s\n", string(body))

				ch <- "Опишите вашу проблему."
			} else {
				fmt.Printf("Код ответа: %d\n", resp.StatusCode)
				var validationErr ValidationError
				if err := json.Unmarshal(body, &validationErr); err != nil {
					fmt.Println(fmt.Errorf("failed to parse validation error: %v", err))
				}
				fmt.Println(validationErr)
				//var validationErr ValidationError
				//if err := json.Unmarshal(resp.Body, &validationErr); err != nil {
				//	fmt.Println(fmt.Errorf("failed to parse validation error: %v", err))
				//}
				//return "", validationErr
			}
		}(restCh)

		respStr := <-restCh

		close(restCh)

		msgArray := make([]Message, 0, 5)

		moscowLocation := time.FixedZone("MSK", 36060) // Смещение +3 часа от UTC

		moscowTime := time.Now().In(moscowLocation)

		msgArray = append(msgArray, Message{Body: respStr, IsBot: true, Time: moscowTime})

		messages.Mu.Lock()

		messages.Storage[chatID] = msgArray

		messages.Mu.Unlock()

		token, err := jwtToken.New(
			chatID,
			time.Hour*300,
			JWTSecret.Secret(),
		)
		if err != nil {
			log.Error("failed to generate token", sl.Err(err))
		}

		cookie := &http.Cookie{
			Name:     "access_token",
			Value:    token,
			MaxAge:   30000,
			Path:     "/",
			HttpOnly: true,
			Secure:   true, // Оставьте false для локальной отладки без HTTPS
			SameSite: http.SameSiteLaxMode,
		}

		http.SetCookie(w, cookie)

		if err := json.NewEncoder(w).Encode(StartChatResponse{
			Resp: respStr,
		}); err != nil {
			log.Info("Error encoding body: ", sl.Err(err))
		}
	}
}

type MessagesResp struct {
	Messages []Message `json:"messages"`
}

func GetMessangesHandler(
	log *slog.Logger,
	messages *MessagesStorage,
	JWTSecret *JWTsecret.JWTSecret,
) func(
	w http.ResponseWriter, r *http.Request,
) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(`запрос /api/message на получение сообщений`)

		token, err := r.Cookie("access_token")
		if err != nil {
			log.Debug("Error getting token", "error", err)
			http_api.HandleError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		fmt.Println("токен был найден")

		accessToken := token.Value

		chatId, err := jwtToken.VerifyToken(accessToken, JWTSecret.Secret())
		if err != nil {
			log.Debug("Error verify token", "error", err)
			http_api.HandleError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		fmt.Println("chatId - ", chatId)

		messages.Mu.RLock()
		defer messages.Mu.RUnlock()

		value, exists := messages.Storage[chatId]
		if exists {
			fmt.Println("сообщения присутствуют, количество сообщений - ", len(messages.Storage[chatId]))
			if err := json.NewEncoder(w).Encode(
				MessagesResp{
					Messages: value,
				},
			); err != nil {
				http_api.HandleError(
					w,
					http.StatusInternalServerError,
					"Error encoding response",
				)
			}
		} else {
			log.Debug("Chat is not exists", "error", err)
			http_api.HandleError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
	}
}

type ResponseMessage struct {
	Body string    `json:"body"`
	Time time.Time `json:"time"`
}

type ClientResponseEndChatBody struct {
	Message string `json:"summary"`
}

func EndChatHandler(
	log *slog.Logger,
	messages *MessagesStorage,
	JWTSecret *JWTsecret.JWTSecret,
) func(
	w http.ResponseWriter, r *http.Request,
) {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := r.Cookie("access_token")
		if err != nil {
			log.Debug("Error getting token", "error", err)
			http_api.HandleError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		accessToken := token.Value

		chatId, err := jwtToken.VerifyToken(accessToken, JWTSecret.Secret())
		if err != nil {
			log.Debug("Error verify token", "error", err)
			http_api.HandleError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		defer func() {
			messages.Mu.Lock()
			delete(messages.Storage, chatId)
			messages.Mu.Unlock()
		}()

		restCh := make(chan string)

		go func(ch chan string) {
			baseURL := "http://localhost:8000/end_dialog"
			params := url.Values{}
			params.Add("user_id", chatId)

			resp, err := http.Post(fmt.Sprintf("%s?%s", baseURL, params.Encode()), "application/json", nil)
			if err != nil {
				fmt.Println(fmt.Errorf("request failed: %v", err))

				ch <- "Сервер недоступен."

				return
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(fmt.Errorf("failed to read response: %v", err))
			}

			//data := url.Values{}
			//data.Set("user_id", chatId)
			//
			//body := bytes.NewBufferString(data.Encode())
			//
			//req, err := http.NewRequest("POST", "http://localhost:8000/end_dialog", body)
			//if err != nil {
			//	fmt.Printf("Ошибка при создании запроса: %v\n", err)
			//	return
			//}

			//client := &http.Client{}
			//resp, err := client.Do(req)
			//if err != nil {
			//	fmt.Printf("Ошибка при выполнении запроса: %v\n", err)
			//	return
			//}
			//defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				fmt.Printf("Тело ответа:\n%s\n", string(body))

				var response ClientResponseEndChatBody
				if err := json.Unmarshal(body, &response); err != nil {
					fmt.Println(fmt.Errorf("failed to parse validation error: %v", err))
				}
				fmt.Println(response)
				fmt.Printf("Тело ответа:\n%s\n", response.Message)

				ch <- response.Message
			} else {
				var validationErr ValidationError
				if err := json.Unmarshal(body, &validationErr); err != nil {
					fmt.Println(fmt.Errorf("failed to parse validation error: %v", err))
				}
				fmt.Println(validationErr)
				fmt.Printf("Код ответа: %d\n", resp.StatusCode)
			}
		}(restCh)

		respStr := <-restCh

		close(restCh)
		http.SetCookie(
			w, &http.Cookie{
				Name:    "access_token",
				Expires: time.Now(),
			},
		)

		moscowLocation := time.FixedZone("MSK", 36060) // Смещение +3 часа от UTC

		moscowTime := time.Now().In(moscowLocation)

		if err := json.NewEncoder(w).Encode(
			ResponseMessage{
				Body: respStr,
				Time: moscowTime,
			},
		); err != nil {
			http_api.HandleError(
				w,
				http.StatusInternalServerError,
				"Error encoding response",
			)
		}
	}
}

type ClientRequestBody struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

type ClientResponseBody struct {
	Message string `json:"response"`
}

type NewMessageHandlerReq struct {
	Body string `json:"message"`
}

func NewMessageHandler(
	log *slog.Logger,
	messages *MessagesStorage,
	JWTSecret *JWTsecret.JWTSecret,
	path string,
) func(
	w http.ResponseWriter, r *http.Request,
) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("New request: ", path)
		token, err := r.Cookie("access_token")
		if err != nil {
			log.Debug("Error getting token", "error", err)
			http_api.HandleError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		accessToken := token.Value

		chatId, err := jwtToken.VerifyToken(accessToken, JWTSecret.Secret())
		if err != nil {
			log.Debug("Error verify token", "error", err)
			http_api.HandleError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		var req NewMessageHandlerReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Info("Error decoding body: ", sl.Err(err))
			http_api.HandleError(
				w,
				http.StatusBadRequest,
				"Invalid request body",
			)
			return
		}

		restCh := make(chan string)

		go func(ch chan string) {
			requestBody := ClientRequestBody{
				UserID:  chatId,
				Message: req.Body,
			}

			jsonData, err := json.Marshal(requestBody)
			if err != nil {
				fmt.Println("Error marshaling JSON:", err)
				return
			}

			resp, err := http.Post("http://localhost:8000/chat", "application/json", bytes.NewBuffer(jsonData))
			if err != nil {
				fmt.Println("Error sending POST request:", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					fmt.Println(fmt.Errorf("failed to read response: %v", err))
				}
				var response ClientResponseBody
				if err := json.Unmarshal(body, &response); err != nil {
					fmt.Println(fmt.Errorf("failed to parse validation error: %v", err))
				}
				fmt.Println(response)
				fmt.Printf("Тело ответа:\n%s\n", response.Message)

				ch <- response.Message
			} else {
				fmt.Printf("Код ответа: %d\n", resp.StatusCode)
			}
		}(restCh)

		respStr := <-restCh

		close(restCh)

		moscowLocation := time.FixedZone("MSK", 36060) // Смещение +3 часа от UTC

		moscowTime := time.Now().In(moscowLocation)

		messages.Mu.Lock()
		defer messages.Mu.Unlock()

		_, exists := messages.Storage[chatId]
		if exists {
			messages.Storage[chatId] = append(messages.Storage[chatId], Message{Body: req.Body, Time: moscowTime, IsBot: false})
			messages.Storage[chatId] = append(messages.Storage[chatId], Message{Body: respStr, Time: moscowTime, IsBot: true})
		} else {
			log.Debug("Chat is not exists", "error", err)
			http_api.HandleError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		if err := json.NewEncoder(w).Encode(
			ResponseMessage{
				Body: respStr,
				Time: moscowTime,
			},
		); err != nil {
			http_api.HandleError(
				w,
				http.StatusInternalServerError,
				"Error encoding response",
			)
		}

	}
}
