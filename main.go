package main

import (
	"bytes"
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/pkg/errors"
)

const (
	handlerTmplStr = `package handler

import (
	"context"

	"{{.GoPackage}}"
)

type {{.RPCName}}HandlerFunc func(ctx context.Context, req *{{.ServiceName}}.{{.RPCName}}Request) (*{{.ServiceName}}.{{.RPCName}}Response, error)

func {{.RPCName}}() {{.RPCName}}HandlerFunc {
	return func(ctx context.Context, req *{{.ServiceName}}.{{.RPCName}}Request) (*{{.ServiceName}}.{{.RPCName}}Response, error) {
		return &{{.ServiceName}}.{{.RPCName}}Response{}, nil
	}
}
`
	handlerTestTmplStr = `package handler

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"{{.GoPackage}}"
)

type {{.RPCName}}HandlerTestSuite struct {
	suite.Suite

	db      *sql.DB
	sqlMock sqlmock.Sqlmock

	ctrl *gomock.Controller
}

func Test{{.RPCName}}HandlerTestSuite(t *testing.T) {
	suite.Run(t, new({{.RPCName}}HandlerTestSuite))
}

func (s *{{.RPCName}}HandlerTestSuite) SetupSuite() {
	monkey.Patch(time.Now, func() time.Time {
		return time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	})
}

func (s *{{.RPCName}}HandlerTestSuite) SetupTest() {
	db, sqlMock, err := sqlmock.New()
	s.NoError(err)

	s.db = db
	s.sqlMock = sqlMock

	s.ctrl = gomock.NewController(s.T())
}

func (s *{{.RPCName}}HandlerTestSuite) TearDownTest() {
	s.sqlMock.ExpectClose()
	err := s.db.Close()
	if err != nil {
		s.T().Log(err)
	}

	s.ctrl.Finish()
}

func (s *{{.RPCName}}HandlerTestSuite) TearDownSuite() {
	monkey.Unpatch(time.Now)
}

func (s *{{.RPCName}}HandlerTestSuite) Test{{.RPCName}}() {
	s.Run("success", func() {
		ctx := context.Background()
		req := &{{.ServiceName}}.{{.RPCName}}Request{}

		resp, err := {{.RPCName}}()(ctx, req)

		s.NoError(err)
		s.Equal(&{{.ServiceName}}.{{.RPCName}}Response{}, resp)
	})
}
`
)

var (
	goPackagePattern   = regexp.MustCompile(`option go_package = "(.*)";`)
	serviceNamePattern = regexp.MustCompile(`service ([A-Z]\w*) {`)
	rpcNamePattern     = regexp.MustCompile(`rpc (\w*)\s?\(.*\)\s?returns.*`)

	matchFirstCap = regexp.MustCompile(`(.)([A-Z][a-z]+)`)
	matchAllCap   = regexp.MustCompile(`([a-z0-9])([A-Z])`)
)

type proto struct {
	GoPackage   string
	ServiceName string
	RPCNames    []string
}

type handler struct {
	GoPackage   string
	ServiceName string
	RPCName     string
}

func parseProto(r io.Reader) (proto, error) {
	var (
		tpl bytes.Buffer

		p proto
	)

	_, err := tpl.ReadFrom(r)
	if err != nil {
		return p, err
	}
	s := tpl.String()

	cand := goPackagePattern.FindStringSubmatch(s)
	if len(cand) == 0 {
		return p, errors.New("no go package option")
	}
	p.GoPackage = cand[1]

	cand = serviceNamePattern.FindStringSubmatch(s)
	if len(cand) == 0 {
		return p, errors.New("no service name")
	}
	p.ServiceName = strings.ToLower(cand[1])

	cands := rpcNamePattern.FindAllStringSubmatch(s, -1)
	if len(cands) == 0 {
		return p, errors.New("no rpcs")
	}

	rpcs := make([]string, len(cands))
	for i, c := range cands {
		rpcs[i] = c[1]
	}
	p.RPCNames = rpcs

	return p, nil
}

func write(w io.Writer, tmplStr string, h handler) error {
	tmpl, err := template.New("handler").Parse(tmplStr)
	if err != nil {
		return err
	}

	code, err := executeTemplate(tmpl, h)
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(code))
	if err != nil {
		return err
	}
	return nil
}

func executeTemplate(t *template.Template, h handler) (string, error) {
	var tpl bytes.Buffer

	err := t.Execute(&tpl, h)
	if err != nil {
		return "", err
	}

	return tpl.String(), nil
}

// refers https://gist.github.com/stoewer/fbe273b711e6a06315d19552dd4d33e6
func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func generateFiles(protoPath, dirPath string) error {
	r, err := os.Open(protoPath)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()

	p, err := parseProto(r)
	if err != nil {
		return err
	}

	for _, rpc := range p.RPCNames {
		h := handler{
			GoPackage:   p.GoPackage,
			ServiceName: p.ServiceName,
			RPCName:     rpc,
		}
		f, err := os.Create(filepath.Join(dirPath, toSnakeCase(rpc)+"_handler.go"))
		if err != nil {
			return err
		}

		if err := write(f, handlerTmplStr, h); err != nil {
			return err
		}

		if err := f.Close(); err != nil {
			return err
		}

		f, err = os.Create(filepath.Join(dirPath, toSnakeCase(rpc)+"_handler_test.go"))
		if err != nil {
			return err
		}

		if err := write(f, handlerTestTmplStr, h); err != nil {
			return err
		}

		if err := f.Close(); err != nil {
			return err
		}

	}
	return nil
}

func main() {
	inputPath := flag.String("i", "", "protobuf file path to write")
	outputDir := flag.String("o", "", "write output to <dir>")
	flag.Parse()

	if *inputPath == "" {
		log.Fatal("missing '-i' flag: provide protobuf file")
	}
	if *outputDir == "" {
		log.Fatal("missing '-o' flag: set output directory")
	}

	if err := generateFiles(*inputPath, *outputDir); err != nil {
		log.Fatal(err)
	}
}
