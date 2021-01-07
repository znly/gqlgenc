package clientgen

import (
	"bytes"
	"fmt"
	"go/types"

	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/Yamashou/gqlgenc/config"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
	"golang.org/x/xerrors"
)

type Source struct {
	schema          *ast.Schema
	queryDocument   *ast.QueryDocument
	sourceGenerator *SourceGenerator
	generateConfig  *config.GenerateConfig
}

func NewSource(schema *ast.Schema, queryDocument *ast.QueryDocument, sourceGenerator *SourceGenerator, generateConfig *config.GenerateConfig) *Source {
	return &Source{
		schema:          schema,
		queryDocument:   queryDocument,
		sourceGenerator: sourceGenerator,
		generateConfig:  generateConfig,
	}
}

type Fragment struct {
	Name string
	Type types.Type
}

func (s *Source) Fragments() ([]*Fragment, error) {
	fragments := make([]*Fragment, 0, len(s.queryDocument.Fragments))
	for _, fragment := range s.queryDocument.Fragments {
		responseFields := s.sourceGenerator.NewResponseFields(fragment.SelectionSet)
		if s.sourceGenerator.cfg.Models.Exists(fragment.Name) {
			return nil, xerrors.New(fmt.Sprintf("%s is duplicated", fragment.Name))
		}

		fragment := &Fragment{
			Name: fragment.Name,
			Type: responseFields.StructType(),
		}

		fragments = append(fragments, fragment)
	}

	for _, fragment := range fragments {
		name := fragment.Name
		s.sourceGenerator.cfg.Models.Add(
			name,
			fmt.Sprintf("%s.%s", s.sourceGenerator.client.Pkg(), templates.ToGo(name)),
		)
	}

	return fragments, nil
}

type Operation struct {
	Name                string
	ResponseStructName  string
	Operation           string
	Args                []*Argument
	VariableDefinitions ast.VariableDefinitionList
	Kind                ast.Operation
}

func NewOperation(operation *ast.OperationDefinition, queryDocument *ast.QueryDocument, args []*Argument, generateConfig *config.GenerateConfig) *Operation {
	return &Operation{
		Name:                operation.Name,
		ResponseStructName:  getResponseStructName(operation, generateConfig),
		Operation:           queryString(queryDocument),
		Args:                args,
		VariableDefinitions: operation.VariableDefinitions,
		Kind:                operation.Operation,
	}
}

func (s *Source) Operations(queryDocuments []*ast.QueryDocument) []*Operation {
	operations := make([]*Operation, 0, len(s.queryDocument.Operations))

	queryDocumentsMap := queryDocumentMapByOperationName(queryDocuments)
	operationArgsMap := s.operationArgsMapByOperationName()
	for _, operation := range s.queryDocument.Operations {
		queryDocument := queryDocumentsMap[operation.Name]
		args := operationArgsMap[operation.Name]
		operations = append(operations, NewOperation(
			operation,
			queryDocument,
			args,
			s.generateConfig,
		))
	}

	return operations
}

func (s *Source) operationArgsMapByOperationName() map[string][]*Argument {
	operationArgsMap := make(map[string][]*Argument)
	for _, operation := range s.queryDocument.Operations {
		operationArgsMap[operation.Name] = s.sourceGenerator.OperationArguments(operation.VariableDefinitions)
	}

	return operationArgsMap
}

func queryDocumentMapByOperationName(queryDocuments []*ast.QueryDocument) map[string]*ast.QueryDocument {
	queryDocumentMap := make(map[string]*ast.QueryDocument)
	for _, queryDocument := range queryDocuments {
		operation := queryDocument.Operations[0]
		queryDocumentMap[operation.Name] = queryDocument
	}

	return queryDocumentMap
}

func queryString(queryDocument *ast.QueryDocument) string {
	var buf bytes.Buffer
	astFormatter := formatter.NewFormatter(&buf)
	astFormatter.FormatQueryDocument(queryDocument)

	return buf.String()
}

type OperationResponse struct {
	Name string
	Type types.Type
}

func (s *Source) OperationResponses() ([]*OperationResponse, error) {
	operationResponse := make([]*OperationResponse, 0, len(s.queryDocument.Operations))
	for _, operation := range s.queryDocument.Operations {
		responseFields := s.sourceGenerator.NewResponseFields(operation.SelectionSet)
		name := getResponseStructName(operation, s.generateConfig)
		if s.sourceGenerator.cfg.Models.Exists(name) {
			return nil, xerrors.New(fmt.Sprintf("%s is duplicated", name))
		}
		operationResponse = append(operationResponse, &OperationResponse{
			Name: name,
			Type: responseFields.StructType(),
		})
	}

	for _, operationResponse := range operationResponse {
		name := operationResponse.Name
		s.sourceGenerator.cfg.Models.Add(
			name,
			fmt.Sprintf("%s.%s", s.sourceGenerator.client.Pkg(), templates.ToGo(name)),
		)
	}

	return operationResponse, nil
}

type Query struct {
	Name string
	Type types.Type
}

func (s *Source) Query() (*Query, error) {
	astDef := s.schema.Query

	if astDef == nil {
		return nil, nil
	}

	fields, err := s.sourceGenerator.NewResponseFieldsByDefinition(astDef)
	if err != nil {
		return nil, xerrors.Errorf("generate failed for Query struct type : %w", err)
	}

	s.sourceGenerator.cfg.Models.Add(
		astDef.Name,
		fmt.Sprintf("%s.%s", s.sourceGenerator.client.Pkg(), templates.ToGo(astDef.Name)),
	)

	return &Query{
		Name: astDef.Name,
		Type: fields.StructType(),
	}, nil
}

type Mutation struct {
	Name string
	Type types.Type
}

func (s *Source) Mutation() (*Mutation, error) {
	astDef := s.schema.Mutation

	if astDef == nil {
		return nil, nil
	}

	fields, err := s.sourceGenerator.NewResponseFieldsByDefinition(astDef)
	if err != nil {
		return nil, xerrors.Errorf("generate failed for Mutation struct type : %w", err)
	}

	s.sourceGenerator.cfg.Models.Add(
		astDef.Name,
		fmt.Sprintf("%s.%s", s.sourceGenerator.client.Pkg(), templates.ToGo(astDef.Name)),
	)

	return &Mutation{
		Name: astDef.Name,
		Type: fields.StructType(),
	}, nil
}

type Subscription struct {
	Name string
	Type types.Type
}

func (s *Source) Subscription() (*Subscription, error) {
	astDef := s.schema.Subscription

	if astDef == nil {
		return nil, nil
	}

	fields, err := s.sourceGenerator.NewResponseFieldsByDefinition(astDef)
	if err != nil {
		return nil, xerrors.Errorf("generate failed for Subscription struct type : %w", err)
	}

	s.sourceGenerator.cfg.Models.Add(
		astDef.Name,
		fmt.Sprintf("%s.%s", s.sourceGenerator.client.Pkg(), templates.ToGo(astDef.Name)),
	)

	return &Subscription{
		Name: astDef.Name,
		Type: fields.StructType(),
	}, nil
}

func getResponseStructName(operation *ast.OperationDefinition, generateConfig *config.GenerateConfig) string {
	name := operation.Name
	if generateConfig != nil {
		if generateConfig.Prefix != nil {
			if operation.Operation == ast.Mutation {
				name = fmt.Sprintf("%s%s", generateConfig.Prefix.Mutation, name)
			}

			if operation.Operation == ast.Query {
				name = fmt.Sprintf("%s%s", generateConfig.Prefix.Query, name)
			}
		}

		if generateConfig.Suffix != nil {
			if operation.Operation == ast.Mutation {
				name = fmt.Sprintf("%s%s", name, generateConfig.Suffix.Mutation)
			}

			if operation.Operation == ast.Query {
				name = fmt.Sprintf("%s%s", name, generateConfig.Suffix.Query)
			}
		}
	}

	return name
}
