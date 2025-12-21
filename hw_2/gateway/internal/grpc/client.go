package grpc

import (
	"context"
	"fmt"
	"os"
	"time"

	"ledger"
	"ledger/proto"
	"ledger/shared"
	"gateway/internal/grpcjson"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)


type Client struct {
	conn   *grpc.ClientConn
	client proto.LedgerServiceClient
}

func NewClient(ctx context.Context) (*Client, func() error, error) {
	addr := getEnvOrDefault("LEDGER_ADDR", "ledger:50052")

	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(dialCtx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(grpcjson.Codec{})),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial ledger: %w", err)
	}

	client := proto.NewLedgerServiceClient(conn)

	closeFn := func() error {
		return conn.Close()
	}

	return &Client{
		conn:   conn,
		client: client,
	}, closeFn, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *Client) SetBudget(ctx context.Context, userID string, budget ledger.Budget) (ledger.Budget, error) {
	req := &proto.SetBudgetRequest{
		UserID: userID,
		Budget: &proto.Budget{
			Category: budget.Category,
			Limit:    budget.Limit,
		},
	}

	resp, err := c.client.SetBudget(ctx, req)
	if err != nil {
		return ledger.Budget{}, mapGRPCError(err)
	}

	return ledger.Budget{
		Category:  resp.Budget.Category,
		Limit:     resp.Budget.Limit,
		Remaining: resp.Budget.Remaining,
		Period:    resp.Budget.Period,
	}, nil
}

func (c *Client) ListBudgets(ctx context.Context, userID string) ([]ledger.Budget, error) {
	req := &proto.ListBudgetsRequest{
		UserID: userID,
	}

	resp, err := c.client.ListBudgets(ctx, req)
	if err != nil {
		return nil, mapGRPCError(err)
	}

	budgets := make([]ledger.Budget, 0, len(resp.Budgets))
	for _, b := range resp.Budgets {
		budgets = append(budgets, ledger.Budget{
			Category:  b.Category,
			Limit:     b.Limit,
			Remaining: b.Remaining,
			Period:    b.Period,
		})
	}

	return budgets, nil
}

func (c *Client) AddTransaction(ctx context.Context, userID string, tx ledger.Transaction) (ledger.Transaction, error) {
	req := &proto.AddTransactionRequest{
		UserID: userID,
		Transaction: &proto.Transaction{
			Amount:      tx.Amount,
			Category:    tx.Category,
			Description: tx.Description,
			Date:        tx.Date,
		},
	}

	resp, err := c.client.AddTransaction(ctx, req)
	if err != nil {
		return ledger.Transaction{}, mapGRPCError(err)
	}

	return ledger.Transaction{
		ID:          int(resp.Transaction.Id),
		Amount:      resp.Transaction.Amount,
		Category:    resp.Transaction.Category,
		Description: resp.Transaction.Description,
		Date:        resp.Transaction.Date,
	}, nil
}

func (c *Client) ListTransactions(ctx context.Context, userID string) ([]ledger.Transaction, error) {
	req := &proto.ListTransactionsRequest{
		UserID: userID,
	}

	resp, err := c.client.ListTransactions(ctx, req)
	if err != nil {
		return nil, mapGRPCError(err)
	}

	transactions := make([]ledger.Transaction, 0, len(resp.Transactions))
	for _, t := range resp.Transactions {
		transactions = append(transactions, ledger.Transaction{
			ID:          int(t.Id),
			Amount:      t.Amount,
			Category:    t.Category,
			Description: t.Description,
			Date:        t.Date,
		})
	}

	return transactions, nil
}

func (c *Client) GetReportSummary(ctx context.Context, userID, from, to string) ([]ledger.ReportSummary, error) {
	req := &proto.GetReportSummaryRequest{
		UserID: userID,
		From:   from,
		To:     to,
	}

	resp, err := c.client.GetReportSummary(ctx, req)
	if err != nil {
		return nil, mapGRPCError(err)
	}

	summary := make([]ledger.ReportSummary, 0, len(resp.Summaries))
	for _, s := range resp.Summaries {
		summary = append(summary, ledger.ReportSummary{
			Category: s.Category,
			Total:    s.Total,
		})
	}

	return summary, nil
}

func (c *Client) ImportTransactionsBulk(ctx context.Context, userID string, txs []ledger.Transaction, workers int) (shared.BulkImportResult, error) {
	protoTxs := make([]*proto.Transaction, 0, len(txs))
	for _, tx := range txs {
		protoTxs = append(protoTxs, &proto.Transaction{
			Amount:      tx.Amount,
			Category:    tx.Category,
			Description: tx.Description,
			Date:        tx.Date,
		})
	}

	req := &proto.ImportTransactionsBulkRequest{
		UserID:       userID,
		Transactions: protoTxs,
		Workers:      int32(workers),
	}

	resp, err := c.client.ImportTransactionsBulk(ctx, req)
	if err != nil {
		return shared.BulkImportResult{}, mapGRPCError(err)
	}

	errors := make([]shared.BulkImportError, 0, len(resp.Result.Errors))
	for _, e := range resp.Result.Errors {
		errors = append(errors, shared.BulkImportError{
			Index: int(e.Index),
			Error: e.Error,
		})
	}

	return shared.BulkImportResult{
		Accepted: int(resp.Result.Accepted),
		Rejected: int(resp.Result.Rejected),
		Errors:   errors,
	}, nil
}

func (c *Client) ImportCSV(ctx context.Context, userID string, csvData string) (shared.BulkImportResult, error) {
	req := &proto.ImportCSVRequest{
		UserID:  userID,
		CsvData: csvData,
	}

	resp, err := c.client.ImportCSV(ctx, req)
	if err != nil {
		return shared.BulkImportResult{}, mapGRPCError(err)
	}

	errors := make([]shared.BulkImportError, 0, len(resp.Result.Errors))
	for _, e := range resp.Result.Errors {
		errors = append(errors, shared.BulkImportError{
			Index: int(e.Index),
			Error: e.Error,
		})
	}

	return shared.BulkImportResult{
		Accepted: int(resp.Result.Accepted),
		Rejected: int(resp.Result.Rejected),
		Errors:   errors,
	}, nil
}

func (c *Client) ExportCSV(ctx context.Context, userID string, from, to string) (string, error) {
	req := &proto.ExportCSVRequest{
		UserID: userID,
		From:   from,
		To:     to,
	}

	resp, err := c.client.ExportCSV(ctx, req)
	if err != nil {
		return "", mapGRPCError(err)
	}

	return resp.CsvData, nil
}

func mapGRPCError(err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return fmt.Errorf("grpc error: %w", err)
	}

	switch st.Code() {
	case codes.FailedPrecondition:
		if st.Message() == "budget exceeded" {
			return ledger.ErrBudgetExceeded
		}
		return fmt.Errorf("failed precondition: %s", st.Message())
	case codes.NotFound:
		return fmt.Errorf("not found: %s", st.Message())
	case codes.InvalidArgument:
		return fmt.Errorf("invalid argument: %s", st.Message())
	case codes.DeadlineExceeded:
		return context.DeadlineExceeded
	case codes.Canceled:
		return context.Canceled
	case codes.Internal:
		return fmt.Errorf("internal error: %s", st.Message())
	default:
		return fmt.Errorf("grpc error [%s]: %s", st.Code(), st.Message())
	}
}

