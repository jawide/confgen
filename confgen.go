package main

import (
	"flag"
	"fmt"
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/spf13/viper"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
)

var (
	targetConfigFile = flag.String("t", "", "targetFile will use this path to read the configuration file")
)

func Usage() {
	_, _ = fmt.Fprintf(os.Stderr, "Description:\n")
	_, _ = fmt.Fprintf(os.Stderr, "\tParsing configFile generates go statement,\n")
	_, _ = fmt.Fprintf(os.Stderr, "\tthen written it args[1] '// config vars' and '// * config func'.\n")
	_, _ = fmt.Fprintf(os.Stderr, "\tThe targetFile is then overwritten\n")
	_, _ = fmt.Fprintf(os.Stderr, "Usage of confgen:\n")
	_, _ = fmt.Fprintf(os.Stderr, "\tconfgen [flags] configFile targetFile\n")
	_, _ = fmt.Fprintf(os.Stderr, "For more information, see:\n")
	_, _ = fmt.Fprintf(os.Stderr, "\thttps://gitee.com/jawide/confgen\n")
	_, _ = fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("confgen: ")
	flag.Usage = Usage
	flag.Parse()
	args := flag.Args()
	if len(args) != 2 {
		Usage()
		return
	}
	v := viper.New()
	v.SetConfigName(strings.Split(filepath.Base(args[0]), ".")[0])
	v.SetConfigType(filepath.Ext(args[0])[1:])
	v.AddConfigPath(filepath.Dir(args[0]))
	err := v.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}

	fs := token.NewFileSet()
	file, _ := decorator.ParseFile(fs, args[1], nil, parser.AllErrors|parser.ParseComments)
	dst.Inspect(file, func(node dst.Node) bool {
		switch n := node.(type) {
		case *dst.GenDecl:
			if n.Lparen && n.Rparen {
				for _, text := range n.Decs.Start {
					if regexp.MustCompile(`^//\s*config\s+vars\s*$`).MatchString(text) {
						n.Specs = []dst.Spec{}
						for _, s := range v.AllKeys() {
							n.Specs = append(n.Specs, &dst.ValueSpec{
								Names: []*dst.Ident{dst.NewIdent(strings.ToUpper(s))},
								Type:  dst.NewIdent(reflect.TypeOf(v.Get(s)).Name()),
							})
						}
						break
					}
				}
			}
			break
		case *dst.FuncDecl:
			for _, text := range n.Decs.Start {
				if regexp.MustCompile(`^//\s*\w+\s+config\s+func\s*$`).MatchString(text) {
					p := args[0]
					if len(*targetConfigFile) > 0 {
						p = *targetConfigFile
					}
					n.Body.List = []dst.Stmt{
						&dst.AssignStmt{
							Lhs: []dst.Expr{dst.NewIdent("v")},
							Tok: token.DEFINE,
							Rhs: []dst.Expr{
								&dst.CallExpr{
									Fun: &dst.SelectorExpr{
										X:   dst.NewIdent("viper"),
										Sel: dst.NewIdent("New")},
								},
							},
							Decs: dst.AssignStmtDecorations{
								NodeDecs: dst.NodeDecs{
									Before: dst.NewLine,
									After:  dst.NewLine,
								},
							},
						},
						&dst.ExprStmt{
							X: &dst.CallExpr{
								Fun: &dst.SelectorExpr{
									X:   dst.NewIdent("v"),
									Sel: dst.NewIdent("SetConfigName"),
								},
								Args: []dst.Expr{
									&dst.BasicLit{
										Kind:  token.STRING,
										Value: "`" + strings.Split(filepath.Base(p), ".")[0] + "`",
									},
								},
							},
							Decs: dst.ExprStmtDecorations{
								NodeDecs: dst.NodeDecs{
									Before: dst.NewLine,
									After:  dst.NewLine,
								},
							},
						},
						&dst.ExprStmt{
							X: &dst.CallExpr{
								Fun: &dst.SelectorExpr{
									X:   dst.NewIdent("v"),
									Sel: dst.NewIdent("SetConfigType"),
								},
								Args: []dst.Expr{
									&dst.BasicLit{
										Kind:  token.STRING,
										Value: "`" + filepath.Ext(p)[1:] + "`",
									},
								},
							},
							Decs: dst.ExprStmtDecorations{
								NodeDecs: dst.NodeDecs{
									Before: dst.NewLine,
									After:  dst.NewLine,
								},
							},
						},
						&dst.ExprStmt{
							X: &dst.CallExpr{
								Fun: &dst.SelectorExpr{
									X:   dst.NewIdent("v"),
									Sel: dst.NewIdent("AddConfigPath"),
								},
								Args: []dst.Expr{
									&dst.BasicLit{
										Kind:  token.STRING,
										Value: "`" + filepath.Dir(p) + "`",
									},
								},
							},
							Decs: dst.ExprStmtDecorations{
								NodeDecs: dst.NodeDecs{
									Before: dst.NewLine,
									After:  dst.NewLine,
								},
							},
						},
						&dst.AssignStmt{
							Lhs: []dst.Expr{dst.NewIdent("err")},
							Tok: token.DEFINE,
							Rhs: []dst.Expr{
								&dst.CallExpr{
									Fun: &dst.SelectorExpr{
										X:   dst.NewIdent("v"),
										Sel: dst.NewIdent("ReadInConfig")},
								},
							},
							Decs: dst.AssignStmtDecorations{
								NodeDecs: dst.NodeDecs{
									Before: dst.NewLine,
									After:  dst.NewLine,
								},
							},
						},
						&dst.IfStmt{
							Cond: &dst.BinaryExpr{
								X:  dst.NewIdent("err"),
								Op: token.NEQ,
								Y:  dst.NewIdent("nil"),
							},
							Body: &dst.BlockStmt{
								List: []dst.Stmt{
									&dst.ExprStmt{
										X: &dst.CallExpr{
											Fun: dst.NewIdent("panic"),
											Args: []dst.Expr{
												dst.NewIdent("err"),
											},
										},
									},
								},
							},
						},
					}
					for _, s := range v.AllKeys() {
						t := reflect.TypeOf(v.Get(s)).Name()
						t = strings.ToUpper(t[:1]) + t[1:]
						n.Body.List = append(n.Body.List, &dst.AssignStmt{
							Lhs: []dst.Expr{dst.NewIdent(strings.ToUpper(s))},
							Tok: token.ASSIGN,
							Rhs: []dst.Expr{
								&dst.CallExpr{
									Fun: &dst.SelectorExpr{
										X:   dst.NewIdent("v"),
										Sel: dst.NewIdent("Get" + t),
									},
									Args: []dst.Expr{
										&dst.BasicLit{
											Kind:  token.STRING,
											Value: "\"" + s + "\"",
										},
									},
								},
							},
							Decs: dst.AssignStmtDecorations{
								NodeDecs: dst.NodeDecs{
									Before: dst.NewLine,
									After:  dst.NewLine,
								},
							},
						})
					}
				}
			}
			break
		}
		return true
	})
	f, err := os.OpenFile(args[1], os.O_WRONLY, 2)
	if err != nil {
		log.Fatal(err)
	}
	err = f.Truncate(0)
	if err != nil {
		log.Fatal(err)
	}
	err = decorator.Fprint(f, file)
	if err != nil {
		log.Fatal(err)
	}
	err = f.Close()
	if err != nil {
		log.Fatal(err)
	}
}
