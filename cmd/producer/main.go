package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/google/uuid"
)

type EventEnvelope struct {
	EventType string      `json:"event_type"`
	Payload   interface{} `json:"payload"`
}

type AccountCreated struct {
	AccountID   string `json:"account_id"`
	CustomerID  string `json:"customer_id"`
	AccountType string `json:"account_type"`
	Currency    string `json:"currency"`
	CreatedAt   string `json:"created_at"`
}

type TransactionAuthorized struct {
	TransactionID   string  `json:"transaction_id"`
	AccountID       string  `json:"account_id"`
	Amount          float64 `json:"amount"`
	TransactionType string  `json:"transaction_type"`
	Method          string  `json:"method"`
	Timestamp       string  `json:"timestamp"`
}

type CardIssued struct {
	CardID         string `json:"card_id"`
	AccountID      string `json:"account_id"`
	CardType       string `json:"card_type"`
	LastFourDigits string `json:"last_four_digits"`
	ExpirationDate string `json:"expiration_date"`
	IsVirtual      bool   `json:"is_virtual"`
}

type CreditAnalysisApproved struct {
	AnalysisID    string `json:"analysis_id"`
	CustomerID    string `json:"customer_id"`
	ApprovedLimit int    `json:"approved_limit"`
	RiskScore     int    `json:"risk_score"`
	ProductType   string `json:"product_type"`
	ValidUntil    string `json:"valid_until"`
}

func randomStringFrom(options []string) string {
	return options[rand.Intn(len(options))]
}

func generateRandomEvent() EventEnvelope {
	eventTypes := []string{
		"ACCOUNT_CREATED",
		"TRANSACTION_AUTHORIZED",
		"CARD_ISSUED",
		"CREDIT_ANALYSIS_APPROVED",
	}
	selectedType := randomStringFrom(eventTypes)
	now := time.Now().UTC()

	var payload interface{}

	switch selectedType {
	case "ACCOUNT_CREATED":
		payload = AccountCreated{
			AccountID:   uuid.NewString(),
			CustomerID:  uuid.NewString(),
			AccountType: randomStringFrom([]string{"CHECKING", "SAVINGS", "SALARY"}),
			Currency:    "BRL",
			CreatedAt:   now.Format(time.RFC3339),
		}
	case "TRANSACTION_AUTHORIZED":
		payload = TransactionAuthorized{
			TransactionID:   uuid.NewString(),
			AccountID:       uuid.NewString(),
			Amount:          math.Round((rand.Float64()*5000+10)*100) / 100,
			TransactionType: randomStringFrom([]string{"CREDIT", "DEBIT"}),
			Method:          randomStringFrom([]string{"PIX", "TED", "CREDIT_CARD", "DEBIT_CARD"}),
			Timestamp:       now.Format(time.RFC3339),
		}
	case "CARD_ISSUED":
		month := rand.Intn(12) + 1
		year := rand.Intn(10) + 25
		payload = CardIssued{
			CardID:         uuid.NewString(),
			AccountID:      uuid.NewString(),
			CardType:       randomStringFrom([]string{"CREDIT", "DEBIT", "MULTIPLE"}),
			LastFourDigits: fmt.Sprintf("%04d", rand.Intn(10000)),
			ExpirationDate: fmt.Sprintf("%02d/%02d", month, year),
			IsVirtual:      rand.Intn(2) == 1,
		}
	case "CREDIT_ANALYSIS_APPROVED":
		payload = CreditAnalysisApproved{
			AnalysisID:    uuid.NewString(),
			CustomerID:    uuid.NewString(),
			ApprovedLimit: rand.Intn(50000) + 1000,
			RiskScore:     rand.Intn(1001),
			ProductType:   randomStringFrom([]string{"PERSONAL_LOAN", "CREDIT_CARD", "MORTGAGE"}),
			ValidUntil:    now.AddDate(0, 3, 0).Format("2006-01-02"),
		}
	}

	return EventEnvelope{
		EventType: selectedType,
		Payload:   payload,
	}
}
func main() {

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("error loading aws config: %v", err)
	}

	client := sns.NewFromConfig(cfg)

	topicARN := os.Getenv("TOPIC_ARN")
	if topicARN == "" {
		log.Fatal("missing env var 'TOPIC_ARN'")
	}

	eventsPerSecond := 10
	if raw := os.Getenv("EVENTS_PER_SECOND"); raw != "" {
		var parsed int
		_, err := fmt.Sscanf(raw, "%d", &parsed)
		if err != nil || parsed <= 0 {
			log.Fatalf("invalid EVENTS_PER_SECOND value: %q", raw)
		}
		eventsPerSecond = parsed
	}

	numWorkers := 10

	eventChannel := make(chan EventEnvelope, 500)
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go snsWorker(ctx, client, topicARN, eventChannel, &wg)
	}

	interval := time.Second / time.Duration(eventsPerSecond)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("publishing continuously at %d events/second. Press Ctrl+C to stop.", eventsPerSecond)

	for {
		select {
		case <-ctx.Done():
			log.Println("stopped by user.")
			close(eventChannel)
			wg.Wait()
			log.Println("process finished!")
			return
		case <-ticker.C:
			eventChannel <- generateRandomEvent()
		}
	}
}

func snsWorker(ctx context.Context, client *sns.Client, topicARN string, eventChannel <-chan EventEnvelope, wg *sync.WaitGroup) {
	defer wg.Done()
	var batch []types.PublishBatchRequestEntry

	for event := range eventChannel {
		bodyBytes, _ := json.Marshal(event.Payload)

		uuidv7, _ := uuid.NewV7()
		eventID := uuidv7.String()
		clientID := uuid.NewString()
		entry := types.PublishBatchRequestEntry{
			Id:      aws.String(eventID),
			Message: aws.String(string(bodyBytes)),
			MessageAttributes: map[string]types.MessageAttributeValue{
				"event_id": {
					DataType:    aws.String("String"),
					StringValue: aws.String(eventID),
				},
				"client_id": {
					DataType:    aws.String("String"),
					StringValue: aws.String(clientID),
				},
				"event_type": {
					DataType:    aws.String("String"),
					StringValue: aws.String(event.EventType),
				},
			},
		}

		batch = append(batch, entry)

		if len(batch) == 10 {
			sendBatchWithRetry(ctx, client, topicARN, batch)
			batch = nil
		}
	}

	if len(batch) > 0 {
		sendBatchWithRetry(ctx, client, topicARN, batch)
	}
}

func sendBatchWithRetry(ctx context.Context, client *sns.Client, topicArn string, batch []types.PublishBatchRequestEntry) {
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries && len(batch) > 0; attempt++ {
		input := &sns.PublishBatchInput{
			TopicArn:                   aws.String(topicArn),
			PublishBatchRequestEntries: batch,
		}

		resp, err := client.PublishBatch(ctx, input)
		if err != nil {
			log.Printf("attempt %d: error sending batch: %v", attempt, err)
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		if len(resp.Failed) == 0 {
			return
		}

		log.Printf("warning: %d failed messages. Attempt %d of %d", len(resp.Failed), attempt, maxRetries)

		failedIDs := make(map[string]bool)
		for _, failed := range resp.Failed {
			failedIDs[*failed.Id] = true
			log.Printf(" - failed reason: %s (Code: %s)", *failed.Message, *failed.Code)
		}

		var nextBatch []types.PublishBatchRequestEntry
		for _, entry := range batch {
			if failedIDs[*entry.Id] {
				nextBatch = append(nextBatch, entry)
			}
		}

		batch = nextBatch
		time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
	}
}
