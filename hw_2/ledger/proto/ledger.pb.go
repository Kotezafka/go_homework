package proto

import (
	"context"
	"errors"

	"google.golang.org/grpc"
)


type LedgerServiceClient interface {
	SetBudget(ctx context.Context, in *SetBudgetRequest, opts ...grpc.CallOption) (*SetBudgetResponse, error)
	ListBudgets(ctx context.Context, in *ListBudgetsRequest, opts ...grpc.CallOption) (*ListBudgetsResponse, error)
	AddTransaction(ctx context.Context, in *AddTransactionRequest, opts ...grpc.CallOption) (*AddTransactionResponse, error)
	ListTransactions(ctx context.Context, in *ListTransactionsRequest, opts ...grpc.CallOption) (*ListTransactionsResponse, error)
	GetReportSummary(ctx context.Context, in *GetReportSummaryRequest, opts ...grpc.CallOption) (*GetReportSummaryResponse, error)
	ImportTransactionsBulk(ctx context.Context, in *ImportTransactionsBulkRequest, opts ...grpc.CallOption) (*ImportTransactionsBulkResponse, error)
	ImportCSV(ctx context.Context, in *ImportCSVRequest, opts ...grpc.CallOption) (*ImportCSVResponse, error)
	ExportCSV(ctx context.Context, in *ExportCSVRequest, opts ...grpc.CallOption) (*ExportCSVResponse, error)
}

type LedgerServiceServer interface {
	SetBudget(context.Context, *SetBudgetRequest) (*SetBudgetResponse, error)
	ListBudgets(context.Context, *ListBudgetsRequest) (*ListBudgetsResponse, error)
	AddTransaction(context.Context, *AddTransactionRequest) (*AddTransactionResponse, error)
	ListTransactions(context.Context, *ListTransactionsRequest) (*ListTransactionsResponse, error)
	GetReportSummary(context.Context, *GetReportSummaryRequest) (*GetReportSummaryResponse, error)
	ImportTransactionsBulk(context.Context, *ImportTransactionsBulkRequest) (*ImportTransactionsBulkResponse, error)
	ImportCSV(context.Context, *ImportCSVRequest) (*ImportCSVResponse, error)
	ExportCSV(context.Context, *ExportCSVRequest) (*ExportCSVResponse, error)
}

type UnimplementedLedgerServiceServer struct{}

func (UnimplementedLedgerServiceServer) SetBudget(context.Context, *SetBudgetRequest) (*SetBudgetResponse, error) {
	return nil, errors.New("unimplemented")
}
func (UnimplementedLedgerServiceServer) ListBudgets(context.Context, *ListBudgetsRequest) (*ListBudgetsResponse, error) {
	return nil, errors.New("unimplemented")
}
func (UnimplementedLedgerServiceServer) AddTransaction(context.Context, *AddTransactionRequest) (*AddTransactionResponse, error) {
	return nil, errors.New("unimplemented")
}
func (UnimplementedLedgerServiceServer) ListTransactions(context.Context, *ListTransactionsRequest) (*ListTransactionsResponse, error) {
	return nil, errors.New("unimplemented")
}
func (UnimplementedLedgerServiceServer) GetReportSummary(context.Context, *GetReportSummaryRequest) (*GetReportSummaryResponse, error) {
	return nil, errors.New("unimplemented")
}
func (UnimplementedLedgerServiceServer) ImportTransactionsBulk(context.Context, *ImportTransactionsBulkRequest) (*ImportTransactionsBulkResponse, error) {
	return nil, errors.New("unimplemented")
}
func (UnimplementedLedgerServiceServer) ImportCSV(context.Context, *ImportCSVRequest) (*ImportCSVResponse, error) {
	return nil, errors.New("unimplemented")
}
func (UnimplementedLedgerServiceServer) ExportCSV(context.Context, *ExportCSVRequest) (*ExportCSVResponse, error) {
	return nil, errors.New("unimplemented")
}

type Budget struct {
	UserID    string  `json:"user_id"`
	Category  string  `json:"category"`
	Limit     float64 `json:"limit"`
	Remaining float64 `json:"remaining"`
	Period    string  `json:"period"`
}

type Transaction struct {
	Id          int32   `json:"id"`
	UserID      string  `json:"user_id"`
	Amount      float64 `json:"amount"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Date        string  `json:"date"`
}

type ReportSummary struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
}

type BulkImportResult struct {
	Accepted int32              `json:"accepted"`
	Rejected int32              `json:"rejected"`
	Errors   []*BulkImportError `json:"errors"`
}

type BulkImportError struct {
	Index int32  `json:"index"`
	Error string `json:"error"`
}

type SetBudgetRequest struct {
	UserID string   `json:"user_id"`
	Budget *Budget `json:"budget"`
}

type SetBudgetResponse struct {
	Budget *Budget `json:"budget"`
}

type ListBudgetsRequest struct {
	UserID string `json:"user_id"`
}

type ListBudgetsResponse struct {
	Budgets []*Budget `json:"budgets"`
}

type AddTransactionRequest struct {
	UserID      string         `json:"user_id"`
	Transaction *Transaction `json:"transaction"`
}

type AddTransactionResponse struct {
	Transaction *Transaction `json:"transaction"`
}

type ListTransactionsRequest struct {
	UserID string `json:"user_id"`
}

type ListTransactionsResponse struct {
	Transactions []*Transaction `json:"transactions"`
}

type GetReportSummaryRequest struct {
	UserID string `json:"user_id"`
	From   string `json:"from"`
	To     string `json:"to"`
}

type GetReportSummaryResponse struct {
	Summaries           []*ReportSummary `json:"summaries"`
	TotalExpenses       float64          `json:"total_expenses"`
	TotalBudget         float64          `json:"total_budget"`
	BudgetUsagePercent  float64          `json:"budget_usage_percent"`
}

type ImportTransactionsBulkRequest struct {
	UserID       string          `json:"user_id"`
	Transactions []*Transaction `json:"transactions"`
	Workers      int32          `json:"workers"`
}

type ImportTransactionsBulkResponse struct {
	Result *BulkImportResult `json:"result"`
}

type ImportCSVRequest struct {
	UserID  string `json:"user_id"`
	CsvData string `json:"csv_data"`
}

type ImportCSVResponse struct {
	Result *BulkImportResult `json:"result"`
}

type ExportCSVRequest struct {
	UserID string `json:"user_id"`
	From   string `json:"from"`
	To     string `json:"to"`
}

type ExportCSVResponse struct {
	CsvData string `json:"csv_data"`
}

type ledgerServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewLedgerServiceClient(cc grpc.ClientConnInterface) LedgerServiceClient {
	return &ledgerServiceClient{cc: cc}
}

func (c *ledgerServiceClient) SetBudget(ctx context.Context, in *SetBudgetRequest, opts ...grpc.CallOption) (*SetBudgetResponse, error) {
	out := new(SetBudgetResponse)
	if err := c.cc.Invoke(ctx, "/ledger.LedgerService/SetBudget", in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *ledgerServiceClient) ListBudgets(ctx context.Context, in *ListBudgetsRequest, opts ...grpc.CallOption) (*ListBudgetsResponse, error) {
	out := new(ListBudgetsResponse)
	if err := c.cc.Invoke(ctx, "/ledger.LedgerService/ListBudgets", in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *ledgerServiceClient) AddTransaction(ctx context.Context, in *AddTransactionRequest, opts ...grpc.CallOption) (*AddTransactionResponse, error) {
	out := new(AddTransactionResponse)
	if err := c.cc.Invoke(ctx, "/ledger.LedgerService/AddTransaction", in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *ledgerServiceClient) ListTransactions(ctx context.Context, in *ListTransactionsRequest, opts ...grpc.CallOption) (*ListTransactionsResponse, error) {
	out := new(ListTransactionsResponse)
	if err := c.cc.Invoke(ctx, "/ledger.LedgerService/ListTransactions", in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *ledgerServiceClient) GetReportSummary(ctx context.Context, in *GetReportSummaryRequest, opts ...grpc.CallOption) (*GetReportSummaryResponse, error) {
	out := new(GetReportSummaryResponse)
	if err := c.cc.Invoke(ctx, "/ledger.LedgerService/GetReportSummary", in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *ledgerServiceClient) ImportTransactionsBulk(ctx context.Context, in *ImportTransactionsBulkRequest, opts ...grpc.CallOption) (*ImportTransactionsBulkResponse, error) {
	out := new(ImportTransactionsBulkResponse)
	if err := c.cc.Invoke(ctx, "/ledger.LedgerService/ImportTransactionsBulk", in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *ledgerServiceClient) ImportCSV(ctx context.Context, in *ImportCSVRequest, opts ...grpc.CallOption) (*ImportCSVResponse, error) {
	out := new(ImportCSVResponse)
	if err := c.cc.Invoke(ctx, "/ledger.LedgerService/ImportCSV", in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *ledgerServiceClient) ExportCSV(ctx context.Context, in *ExportCSVRequest, opts ...grpc.CallOption) (*ExportCSVResponse, error) {
	out := new(ExportCSVResponse)
	if err := c.cc.Invoke(ctx, "/ledger.LedgerService/ExportCSV", in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

func RegisterLedgerServiceServer(s grpc.ServiceRegistrar, srv LedgerServiceServer) {
	s.RegisterService(&LedgerService_ServiceDesc, srv)
}

func _LedgerService_SetBudget_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetBudgetRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LedgerServiceServer).SetBudget(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ledger.LedgerService/SetBudget",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LedgerServiceServer).SetBudget(ctx, req.(*SetBudgetRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _LedgerService_ListBudgets_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListBudgetsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LedgerServiceServer).ListBudgets(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ledger.LedgerService/ListBudgets",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LedgerServiceServer).ListBudgets(ctx, req.(*ListBudgetsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _LedgerService_AddTransaction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddTransactionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LedgerServiceServer).AddTransaction(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ledger.LedgerService/AddTransaction",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LedgerServiceServer).AddTransaction(ctx, req.(*AddTransactionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _LedgerService_ListTransactions_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListTransactionsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LedgerServiceServer).ListTransactions(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ledger.LedgerService/ListTransactions",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LedgerServiceServer).ListTransactions(ctx, req.(*ListTransactionsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _LedgerService_GetReportSummary_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetReportSummaryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LedgerServiceServer).GetReportSummary(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ledger.LedgerService/GetReportSummary",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LedgerServiceServer).GetReportSummary(ctx, req.(*GetReportSummaryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _LedgerService_ImportTransactionsBulk_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ImportTransactionsBulkRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LedgerServiceServer).ImportTransactionsBulk(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ledger.LedgerService/ImportTransactionsBulk",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LedgerServiceServer).ImportTransactionsBulk(ctx, req.(*ImportTransactionsBulkRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _LedgerService_ImportCSV_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ImportCSVRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LedgerServiceServer).ImportCSV(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ledger.LedgerService/ImportCSV",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LedgerServiceServer).ImportCSV(ctx, req.(*ImportCSVRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _LedgerService_ExportCSV_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ExportCSVRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LedgerServiceServer).ExportCSV(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ledger.LedgerService/ExportCSV",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LedgerServiceServer).ExportCSV(ctx, req.(*ExportCSVRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var LedgerService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "ledger.LedgerService",
	HandlerType: (*LedgerServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "SetBudget", Handler: _LedgerService_SetBudget_Handler},
		{MethodName: "ListBudgets", Handler: _LedgerService_ListBudgets_Handler},
		{MethodName: "AddTransaction", Handler: _LedgerService_AddTransaction_Handler},
		{MethodName: "ListTransactions", Handler: _LedgerService_ListTransactions_Handler},
		{MethodName: "GetReportSummary", Handler: _LedgerService_GetReportSummary_Handler},
		{MethodName: "ImportTransactionsBulk", Handler: _LedgerService_ImportTransactionsBulk_Handler},
		{MethodName: "ImportCSV", Handler: _LedgerService_ImportCSV_Handler},
		{MethodName: "ExportCSV", Handler: _LedgerService_ExportCSV_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "ledger.proto",
}