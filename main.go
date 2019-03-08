package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/parser"
	"golang.org/x/net/html"
)

type Translation struct {
	textSourceLang string
	textTargetLang string
}

func processJS(js string) []Translation {
	program, err := parser.ParseFile(nil, "", js, 0)
	if err != nil {
		panic(err)
	}

	var sourceElements []string
	var targetElements []string
	for _, x := range program.Body {
		vstmnt, ok := x.(*ast.VariableStatement)
		if !ok {
			continue
		}

		for _, expr := range vstmnt.List {
			vexpr, ok := expr.(*ast.VariableExpression)
			if !ok {
				continue
			}
			//source language
			if vexpr.Name == "c1Arr" {
				sourceElements = parseNewExpression(vexpr.Initializer)
			}
			//target language
			if vexpr.Name == "c2Arr" {
				targetElements = parseNewExpression(vexpr.Initializer)
			}
		}

	}

	if len(sourceElements) != len(targetElements) {
		return nil
	}
	translations := make([]Translation, 0, len(sourceElements))
	for i, sourceElement := range sourceElements {
		if sourceElement == "" {
			continue //skip empty elements
		}
		translation := Translation{
			textSourceLang: sourceElement,
			textTargetLang: targetElements[i],
		}
		translations = append(translations, translation)
	}
	return translations
}

func extractJS(body io.Reader) string {
	tokeniser := html.NewTokenizer(body)
	triggered := false
	for {
		tt := tokeniser.Next()
		if tt == html.ErrorToken {
			break
		}
		if triggered && tt == html.TextToken {
			text := string(tokeniser.Text())
			if strings.Contains(text, "var c1Arr = new Array") {
				return text
			}
		}
		if tt == html.StartTagToken && tokeniser.Token().String() == `<script type="text/javascript">` {
			triggered = true
		}
		if tt == html.EndTagToken && tokeniser.Token().String() == "</script>" {
			triggered = false
		}
	}
	return ""
}

func parseNewExpression(node ast.Expression) []string {
	nexpr, ok := node.(*ast.NewExpression)
	if !ok {
		return nil
	}
	elements := make([]string, 0)
	for _, arg := range nexpr.ArgumentList {
		sliteral, ok := arg.(*ast.StringLiteral)
		if !ok {
			continue
		}
		elements = append(elements, sliteral.Value)
	}
	return elements
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Please provide a search term")
	}
	search := strings.Join(os.Args[1:], " ")
	u := "https://www.dict.cc/?s=" + url.QueryEscape(search)
	resp, err := http.Get(u)
	if err != nil {
		panic(err)
	}
	js := extractJS(resp.Body)
	resp.Body.Close()
	translations := processJS(js)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"en", "de"})

	for _, translation := range translations {
		table.Append([]string{translation.textSourceLang, translation.textTargetLang})
	}
	table.Render()
}
