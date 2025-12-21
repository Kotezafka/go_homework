package ledger

import (
	"context"
	"fmt"

	"ledger/internal/domain"
	"ledger/internal/service"
	"ledger/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)


type Server struct {
	proto.UnimplementedLedgerServiceServer
	service service.Service
}

func NewServer(svc service.Service) *Server {
	return &Server{service: svc}
}

func (s *Server) SetBudget(ctx context.Context, req *proto.SetBudgetRequest) (*proto.SetBudgetResponse, error) {
	budget := domain.Budget{
		Category: req.Budget.Category,
		Limit:    req.Budget.Limit,
	}

	result, err := s.service.SetBudget(ctx, req.UserID, budget)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("set budget: %v", err))
	}

	return &proto.SetBudgetResponse{
		Budget: &proto.Budget{
			Category:  result.Category,
			Limit:     result.Limit,
			Remaining: result.Remaining,
			Period:    result.Period,
		},
	}, nil
}

func (s *Server) ListBudgets(ctx context.Context, req *proto.ListBudgetsRequest) (*proto.ListBudgetsResponse, error) {
	budgets, err := s.service.ListBudgets(ctx, req.UserID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("list budgets: %v", err))
	}

	var result []*proto.Budget
	for _, b := range budgets {
		result = append(result, &proto.Budget{
			Category:  b.Category,
			Limit:     b.Limit,
			Remaining: b.Remaining,
			Period:    b.Period,
		})
	}

	return &proto.ListBudgetsResponse{Budgets: result}, nil
}

func (s *Server) AddTransaction(ctx context.Context, req *proto.AddTransactionRequest) (*proto.AddTransactionResponse, error) {
	tx := domain.Transaction{
		Amount:      req.Transaction.Amount,
		Category:    req.Transaction.Category,
		Description: req.Transaction.Description,
		Date:        req.Transaction.Date,
	}

	result, err := s.service.AddTransaction(ctx, req.UserID, tx)
	if err != nil {
		if err == domain.ErrBudgetExceeded {
			return nil, status.Error(codes.FailedPrecondition, "budget exceeded")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("add transaction: %v", err))
	}

	return &proto.AddTransactionResponse{
		Transaction: &proto.Transaction{
			Id:          int32(result.ID),
			Amount:      result.Amount,
			Category:    result.Category,
			Description: result.Description,
			Date:        result.Date,
		},
	}, nil
}

func (s *Server) ListTransactions(ctx context.Context, req *proto.ListTransactionsRequest) (*proto.ListTransactionsResponse, error) {
	transactions, err := s.service.ListTransactions(ctx, req.UserID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("list transactions: %v", err))
	}

	var result []*proto.Transaction
	for _, tx := range transactions {
		result = append(result, &proto.Transaction{
			Id:          int32(tx.ID),
			Amount:      tx.Amount,
			Category:    tx.Category,
			Description: tx.Description,
			Date:        tx.Date,
		})
	}

	return &proto.ListTransactionsResponse{Transactions: result}, nil
}

func (s *Server) GetReportSummary(ctx context.Context, req *proto.GetReportSummaryRequest) (*proto.GetReportSummaryResponse, error) {
	summary, err := s.service.GetReportSummary(ctx, req.UserID, req.From, req.To)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("get report summary: %v", err))
	}

	var result []*proto.ReportSummary
	var totalExpenses float64
	for _, s := range summary {
		result = append(result, &proto.ReportSummary{
			Category: s.Category,
			Total:    s.Total,
		})
		totalExpenses += s.Total
	}

	budgets, err := s.service.ListBudgets(ctx, req.UserID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("get budgets for report: %v", err))
	}

	var totalBudget float64
	for _, b := range budgets {
		totalBudget += b.Limit
	}

	budgetUsagePercent := 0.0
	if totalBudget > 0 {
		budgetUsagePercent = (totalExpenses / totalBudget) * 100
	}

	return &proto.GetReportSummaryResponse{
		Summaries:           result,
		TotalExpenses:       totalExpenses,
		TotalBudget:         totalBudget,
		BudgetUsagePercent:  budgetUsagePercent,
	}, nil
}

func (s *Server) ImportTransactionsBulk(ctx context.Context, req *proto.ImportTransactionsBulkRequest) (*proto.ImportTransactionsBulkResponse, error) {
	var transactions []domain.Transaction
	for _, t := range req.Transactions {
		transactions = append(transactions, domain.Transaction{
			Amount:      t.Amount,
			Category:    t.Category,
			Description: t.Description,
			Date:        t.Date,
		})
	}

	result, err := s.service.ImportTransactionsBulk(ctx, req.UserID, transactions, int(req.Workers))
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("bulk import: %v", err))
	}

	var errors []*proto.BulkImportError
	for _, e := range result.Errors {
		errors = append(errors, &proto.BulkImportError{
			Index: int32(e.Index),
			Error: e.Error,
		})
	}

	return &proto.ImportTransactionsBulkResponse{
		Result: &proto.BulkImportResult{
			Accepted: int32(result.Accepted),
			Rejected: int32(result.Rejected),
			Errors:   errors,
		},
	}, nil
}

func (s *Server) ImportCSV(ctx context.Context, req *proto.ImportCSVRequest) (*proto.ImportCSVResponse, error) {
	result, err := s.service.ImportCSV(ctx, req.UserID, req.CsvData)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("import CSV: %v", err))
	}

	var errors []*proto.BulkImportError
	for _, e := range result.Errors {
		errors = append(errors, &proto.BulkImportError{
			Index: int32(e.Index),
			Error: e.Error,
		})
	}

	return &proto.ImportCSVResponse{
		Result: &proto.BulkImportResult{
			Accepted: int32(result.Accepted),
			Rejected: int32(result.Rejected),
			Errors:   errors,
		},
	}, nil
}

func (s *Server) ExportCSV(ctx context.Context, req *proto.ExportCSVRequest) (*proto.ExportCSVResponse, error) {
	csvData, err := s.service.ExportCSV(ctx, req.UserID, req.From, req.To)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("export CSV: %v", err))
	}

	return &proto.ExportCSVResponse{CsvData: csvData}, nil
}