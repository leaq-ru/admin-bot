package call

import (
	"github.com/leaq-ru/proto/codegen/go/parser"
	"google.golang.org/grpc"
)

func NewClients(parserURL string) (companyClient parser.CompanyClient, reviewClient parser.ReviewClient, err error) {
	connParser, err := grpc.Dial(parserURL, grpc.WithInsecure())
	if err != nil {
		return
	}
	companyClient = parser.NewCompanyClient(connParser)
	reviewClient = parser.NewReviewClient(connParser)
	return
}
