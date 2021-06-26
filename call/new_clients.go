package call

import (
	"github.com/nnqq/scr-proto/codegen/go/parser"
	"google.golang.org/grpc"
)

func NewClients(parserURL string) (companyClient parser.CompanyClient, err error) {
	connParser, err := grpc.Dial(parserURL, grpc.WithInsecure())
	if err != nil {
		return
	}
	companyClient = parser.NewCompanyClient(connParser)
	return
}
