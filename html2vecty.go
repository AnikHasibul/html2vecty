package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"bytes"
	"io"

	"strings"

	"encoding/xml"

	"github.com/aymerick/douceur/parser"
	"github.com/dave/jennifer/jen"
)

const defaultText = `
<h1 id="vecty" class="greetings welcome">
Hello world!
</h1>
`

func NewTranscoder(input string) *SourceCode {
	if strings.TrimSpace(input) == "" {
		input = defaultText
	}
	s := &SourceCode{
		html: strings.TrimSpace(input),
	}
	s.transcode()

	return s
}

type SourceCode struct {
	html, vecty string
}

func (s *SourceCode) Html() string {
	return s.html
}

func (s *SourceCode) Code() string {
	return s.vecty
}
func (s *SourceCode) transcode() error {
	decoder := xml.NewDecoder(bytes.NewBufferString(s.html))

	EOT := errors.New("end of tag")
	call := jen.Options{
		Close:     ")",
		Multi:     true,
		Open:      "(",
		Separator: ",",
	}

	var transcode func(*xml.Decoder) (jen.Code, error)
	transcode = func(decoder *xml.Decoder) (code jen.Code, err error) {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}

		switch token := token.(type) {
		case xml.StartElement:
			tag := token.Name.Local
			vectyFunction, ok := elemNameMap[tag]
			vectyPackage := "github.com/gopherjs/vecty/elem"
			vectyParamater := ""
			if !ok {
				vectyFunction = "Tag"
				vectyPackage = "github.com/gopherjs/vecty"
				vectyParamater = tag
			}
			var outer error
			q := jen.Qual(vectyPackage, vectyFunction).CustomFunc(call, func(g *jen.Group) {
				if vectyParamater != "" {
					g.Lit(vectyParamater)
				}
				if len(token.Attr) > 0 {
					g.Qual("github.com/gopherjs/vecty", "Markup").CustomFunc(call, func(g *jen.Group) {
						for _, v := range token.Attr {
							switch {
							case v.Name.Local == "style":
								css, err := parser.ParseDeclarations(v.Value)
								if err != nil {
									outer = err
									return
								}
								for _, dec := range css {
									if dec.Important {
										dec.Value += "!important"
									}
									g.Qual("github.com/gopherjs/vecty", "Style").Call(
										jen.Lit(dec.Property),
										jen.Lit(dec.Value),
									)
								}
							case v.Name.Local == "class":
								g.Qual("github.com/gopherjs/vecty", "Class").CallFunc(func(g *jen.Group) {
									classes := strings.Split(v.Value, " ")
									for _, class := range classes {
										g.Lit(class)
									}
								})
							case strings.HasPrefix(v.Name.Local, "data-"):
								attribute := strings.TrimPrefix(v.Name.Local, "data-")
								g.Qual("github.com/gopherjs/vecty", "Data").Call(
									jen.Lit(attribute),
									jen.Lit(v.Value),
								)
							case boolProps[v.Name.Local] != "":
								value := v.Value == "true"
								g.Qual("github.com/gopherjs/vecty/prop", boolProps[v.Name.Local]).Call(
									jen.Lit(value),
								)
							case stringProps[v.Name.Local] != "":
								g.Qual("github.com/gopherjs/vecty/prop", stringProps[v.Name.Local]).Call(
									jen.Lit(v.Value),
								)
							case v.Name.Local == "xmlns":
								g.Qual("github.com/gopherjs/vecty", "Namespace").Call(
									jen.Lit(v.Value),
								)
							case v.Name.Local == "type" && typeProps[v.Value] != "":
								g.Qual("github.com/gopherjs/vecty/prop", "Type").Call(
									jen.Qual("github.com/gopherjs/vecty/prop", typeProps[v.Value]),
								)
							default:
								g.Qual("github.com/gopherjs/vecty", "Attribute").Call(
									jen.Lit(v.Name.Local),
									jen.Lit(v.Value),
								)
							}
						}
					})
				}
				for {
					c, err := transcode(decoder)
					if err != nil {
						if err == EOT {
							break
						}
						outer = err
						return
					}
					if c != nil {
						g.Add(c)
					}
				}
			})
			if outer != nil {
				return nil, outer
			}
			return q, nil
		case xml.CharData:
			s := strings.TrimSpace(string(token))
			if s == "" {
				return nil, nil
			}
			return jen.Qual("github.com/gopherjs/vecty", "Text").Call(jen.Lit(s)), nil
		case xml.EndElement:
			return nil, EOT
		default:
			fmt.Printf("%T %#v\n", token, token)
		}
		return nil, nil
	}

	file := jen.NewFile("main")
	file.PackageComment("This file was created with html2vecty (https://github.com/AnikHasibul/html2vecty)")
	file.ImportNames(map[string]string{
		"github.com/gopherjs/vecty":       "vecty",
		"github.com/gopherjs/vecty/elem":  "elem",
		"github.com/gopherjs/vecty/prop":  "prop",
		"github.com/gopherjs/vecty/event": "event",
		"github.com/gopherjs/vecty/style": "style",
	})
	var elements []jen.Code
	for {
		c, err := transcode(decoder)
		if err != nil {
			if err == io.EOF || err == EOT {
				break
			}
			s.vecty = fmt.Sprintf("%s", err)
			return nil
		}
		if c != nil {
			elements = append(elements, c)
		}
	}
	/*
		func main() {
			vecty.RenderBody(&Page{})
		}

		type Page struct {
			vecty.Core
		}

		func (*Page) Render() vecty.ComponentOrHTML {
			return elem.Body(...)
		}
	*/
	file.Func().Id("main").Params().Block(
		jen.Qual("github.com/gopherjs/vecty", "RenderBody").Call(
			jen.Op("&").Id("Page").Values(),
		),
	)
	file.Type().Id("Page").Struct(
		jen.Qual("github.com/gopherjs/vecty", "Core"),
	)
	file.Func().Params(jen.Op("*").Id("Page")).Id("Render").Params().Qual("github.com/gopherjs/vecty", "ComponentOrHTML").Block(
		jen.Return(
			jen.Qual("github.com/gopherjs/vecty/elem", "Body").Custom(call, elements...),
		),
	)
	/*if len(elements) == 1 {
		file.Var().Id("Element").Op("=").Add(elements[0])
	} else if len(elements) > 1 {
		file.Var().Id("Elements").Op("=").Index().Op("*").Qual("github.com/gopherjs/vecty", "HTML").Custom(values, elements...)
	}*/

	buf := &bytes.Buffer{}
	if err := file.Render(buf); err != nil {
		s.vecty = fmt.Sprintf("%s", err)
		return nil
	}

	s.vecty = buf.String()
	return nil
}

var elemNameMap = map[string]string{
	"a":          "Anchor",
	"abbr":       "Abbreviation",
	"address":    "Address",
	"area":       "Area",
	"article":    "Article",
	"aside":      "Aside",
	"audio":      "Audio",
	"b":          "Bold",
	"base":       "Base",
	"bdi":        "BidirectionalIsolation",
	"bdo":        "BidirectionalOverride",
	"blockquote": "BlockQuote",
	"body":       "Body",
	"br":         "Break",
	"button":     "Button",
	"canvas":     "Canvas",
	"caption":    "Caption",
	"cite":       "Citation",
	"code":       "Code",
	"col":        "Column",
	"colgroup":   "ColumnGroup",
	"data":       "Data",
	"datalist":   "DataList",
	"dd":         "Description",
	"del":        "DeletedText",
	"details":    "Details",
	"dfn":        "Definition",
	"dialog":     "Dialog",
	"div":        "Div",
	"dl":         "DescriptionList",
	"dt":         "DefinitionTerm",
	"em":         "Emphasis",
	"embed":      "Embed",
	"fieldset":   "FieldSet",
	"figcaption": "FigureCaption",
	"figure":     "Figure",
	"footer":     "Footer",
	"form":       "Form",
	"h1":         "Heading1",
	"h2":         "Heading2",
	"h3":         "Heading3",
	"h4":         "Heading4",
	"h5":         "Heading5",
	"h6":         "Heading6",
	"header":     "Header",
	"hgroup":     "HeadingsGroup",
	"hr":         "HorizontalRule",
	"i":          "Italic",
	"iframe":     "InlineFrame",
	"img":        "Image",
	"input":      "Input",
	"ins":        "InsertedText",
	"kbd":        "KeyboardInput",
	"label":      "Label",
	"legend":     "Legend",
	"li":         "ListItem",
	"link":       "Link",
	"main":       "Main",
	"map":        "Map",
	"mark":       "Mark",
	"meta":       "Meta",
	"meter":      "Meter",
	"nav":        "Navigation",
	"noscript":   "NoScript",
	"object":     "Object",
	"ol":         "OrderedList",
	"optgroup":   "OptionsGroup",
	"option":     "Option",
	"output":     "Output",
	"p":          "Paragraph",
	"param":      "Parameter",
	"picture":    "Picture",
	"pre":        "Preformatted",
	"progress":   "Progress",
	"q":          "Quote",
	"rp":         "RubyParenthesis",
	"rt":         "RubyText",
	"rtc":        "RubyTextContainer",
	"ruby":       "Ruby",
	"s":          "Strikethrough",
	"samp":       "Sample",
	"script":     "Script",
	"section":    "Section",
	"select":     "Select",
	"slot":       "Slot",
	"small":      "Small",
	"source":     "Source",
	"span":       "Span",
	"strong":     "Strong",
	"style":      "Style",
	"sub":        "Subscript",
	"summary":    "Summary",
	"sup":        "Superscript",
	"table":      "Table",
	"tbody":      "TableBody",
	"td":         "TableData",
	"template":   "Template",
	"textarea":   "TextArea",
	"tfoot":      "TableFoot",
	"th":         "TableHeader",
	"thead":      "TableHead",
	"time":       "Time",
	"title":      "Title",
	"tr":         "TableRow",
	"track":      "Track",
	"u":          "Underline",
	"ul":         "UnorderedList",
	"var":        "Variable",
	"video":      "Video",
	"wbr":        "WordBreakOpportunity",
}

var typeProps = map[string]string{
	"button":         "TypeButton",
	"checkbox":       "TypeCheckbox",
	"color":          "TypeColor",
	"date":           "TypeDate",
	"datetime":       "TypeDatetime",
	"datetime-local": "TypeDatetimeLocal",
	"email":          "TypeEmail",
	"file":           "TypeFile",
	"hidden":         "TypeHidden",
	"image":          "TypeImage",
	"month":          "TypeMonth",
	"number":         "TypeNumber",
	"password":       "TypePassword",
	"radio":          "TypeRadio",
	"range":          "TypeRange",
	"min":            "TypeMin",
	"max":            "TypeMax",
	"value":          "TypeValue",
	"step":           "TypeStep",
	"reset":          "TypeReset",
	"search":         "TypeSearch",
	"submit":         "TypeSubmit",
	"tel":            "TypeTel",
	"text":           "TypeText",
	"time":           "TypeTime",
	"url":            "TypeUrl",
	"week":           "TypeWeek",
}

var boolProps = map[string]string{
	"autofocus": "Autofocus",
	"checked":   "Checked",
}

var stringProps = map[string]string{
	"for":         "For",
	"href":        "Href",
	"id":          "ID",
	"placeholder": "Placeholder",
	"src":         "Src",
	"value":       "Value",
}

func main() {
	var in []byte
	var err error
	if len(os.Args) < 1 {
		in, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else if len(os.Args) == 2 {
		if c := os.Args[1]; c == "--help" || c == "-h" || c == "help" {
			const HELPMSG = `
html2vecty - A tool for transpiling html source code to vecty component!

Example usage:

1. FROM STDIN: html2vecty can transpile your code from stdin. ex:
	$ cat index.html | html2vecty
or
	$ echo '<h1>Hey Vecty</h1>' | html2vecty

2. FROM FILES: html2vecty can transpile your code from files. ex:
	$ html2vecty index.html

By default html2vecty writes output on stdout. So writing to file is easy!
	$ html2vecty index.html > index.go

Done!

For any kind of query, issue, suggestion,

https://github.com/AnikHasibul/html2vecty
anikhasibul@outlook.com

Thank you!
	`
			fmt.Fprintln(os.Stderr, HELPMSG)
			os.Exit(0)
		}
		in, err = ioutil.ReadFile(os.Args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	t := NewTranscoder(string(in))
	fmt.Fprintln(os.Stdout, t.Code())

}
