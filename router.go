package routers

import (
	"baseadmin/src/controllers"
	"errors"
	"fmt"
	"github.com/asktop/gotools/afile"
	"github.com/asktop/gotools/ajson"
	"github.com/asktop/gotools/akey"
	"github.com/astaxie/beego"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

func init() {
	err := generateRouter()
	if err != nil {
		panic(err)
	}
	fmt.Println("generate routers")

	beego.Router("/", &controllers.MainController{})
}

const (
	//生成路由文件的前缀名称
	ROUTER_PREFIX = "commentsRouter_"
	//项目url的统一前缀
	PROJECT_ROUTER_PREFIX = "/v1"
	//源码相对于根目录的存放位置
	PROJECT_SOURCE_CATALOG = "src"
	//项目固定的名称（因想直接修改项目文件夹名称，不去修改go.mod 而项目能正常运行）
	PEOJECT_NAME = "baseadmin"
)

var routeRegex = regexp.MustCompile(`@router\s+(\S+)(?:\s+\[(\S+)\])?`)
var actionRegex = regexp.MustCompile(`@action\s+(\S*)`)

func generateRouter() error {
	//路径是否存在
	path := filepath.Join(getProjectPath(), PROJECT_SOURCE_CATALOG, "routers")
	if !afile.IsExist(path) {
		err := afile.CreateDir(path, os.ModePerm)
		if err != nil {
			return err
		}
	}
	//扫描文件
	actions, err := getActions()
	if err != nil {
		return err
	}
	//校验是否需要重新生成路由
	if !hasChanged(actions) {
		fmt.Println("routers had no change")
		return nil
	}
	deleteRouter(path)

	for _, action := range actions {
		//如果action router为空，并且mentods为空则跳过
		if action == nil || action.Methods == nil || len(action.Methods) == 0 {
			continue
		}
		//路由代码
		code := getRouterCode(action)
		//文件名称
		fileName := ROUTER_PREFIX + action.Package + "_" + action.ControllerName + ".go"
		filePath := filepath.Join(path, fileName)
		err := afile.WriteFile(filePath, code, true)
		if err != nil {
			return err
		}
	}
	return nil
}

//获取路由代码
func getRouterCode(action *Action) string {
	//处理替换项目名称，固定为指定的名称
	tmp := strings.Split(action.Path, "/")
	tmp[0] = PEOJECT_NAME
	action.Path = strings.Join(tmp, "/")

	body := "package routers"
	body += "\n\n"
	body += "import (\n"
	body += "\t\"github.com/astaxie/beego\"\n"
	body += "\t\"" + action.Path + "\"\n"
	body += ")"
	body += "\n\n"

	body += "func init() {\n"
	controllers := "&" + action.Package + "." + action.ControllerName + "{}"
	for _, method := range action.Methods {
		router := PROJECT_ROUTER_PREFIX + action.Router + method.Router
		body += "\tbeego.Router(\"" + router + "\", " + controllers + ", \"" + method.Method + ":" + method.FuncName + "\")\n";
	}
	body += "}"
	return body
}

//路由文件是否更改
func hasChanged(actions []*Action) bool {
	original := ajson.Encode(actions)
	var old_md5 string
	path := filepath.Join(getProjectPath(), "routers.tmp")
	if afile.IsExist(path) {
		old_md5, _ = afile.ReadFile(path)
	}
	new_md5 := akey.Md5(original)
	if new_md5 == old_md5 {
		return false
	}
	afile.WriteFile(path, new_md5, true)
	return true
}

//删除路由文件
func deleteRouter(dir string) {
	_, names, _ := afile.GetNames(dir)
	for _, name := range names {
		if ok, _ := path.Match("commentsRouter_*.go", name); ok {
			afile.Delete(filepath.Join(dir, name))
		}
	}
}

//扫描目录，获取注解
func getActions() ([]*Action, error) {
	projectPath := getProjectPath()
	pkgRealpath := filepath.Join(projectPath, PROJECT_SOURCE_CATALOG, "controllers")
	if !afile.IsExist(pkgRealpath) {
		panic("controllers dir is not exists")
	}
	var actions []*Action
	//获取所有要扫描的包路径
	dirs := getScanDirPath(pkgRealpath)
	for _, dir := range dirs {
		//获取包路径
		pkgpath := getPkgpath(dir)
		tmp, err := parsePkg(dir, pkgpath)
		if err != nil {
			return nil, err
		}
		actions = append(actions, tmp...)
	}

	return actions, nil
}

func getProjectPath() string {
	dir, _ := os.Getwd()
	return dir
}

func getPkgpath(pkgRealpath string) string {
	//获取project目录
	rootPath := filepath.Dir(getProjectPath())
	rootPath = filepath.Join(rootPath, "\\")
	tmp := strings.ReplaceAll(pkgRealpath, rootPath, "")
	if strings.Contains(tmp, "\\") {
		tmp = strings.ReplaceAll(tmp, "\\", "/")
	}
	if strings.HasPrefix(tmp, "/") {
		tmp = strings.TrimLeft(tmp, "/")
	}
	return tmp
}

//获取需要扫描的所有包路径
func getScanDirPath(controllersPath string) []string {
	dirNames, _, _ := afile.GetNames(controllersPath)
	for i, dir := range dirNames {
		dirNames[i] = filepath.Join(controllersPath, dir)
	}
	dirNames = append(dirNames, controllersPath)
	return dirNames
}

//扫描单个文件
func getAction(path string) (*Action, error) {
	//读取文件
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	action := &Action{}
	for _, line := range lines {
		//获取包名
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "package") {
			action.Package = strings.TrimSpace(strings.TrimLeft(t, "package"))
			continue
		}

		//获取action注解
		t = strings.TrimSpace(strings.TrimLeft(line, "//"))
		if strings.HasPrefix(t, "@action") {
			matches := actionRegex.FindStringSubmatch(t)
			if len(matches) == 2 {
				action.Router = matches[1]
			} else {
				return nil, errors.New("action annotate is error in file " + path)
			}
			return action, nil
		}
	}
	return action, nil
}

type Action struct {
	Path           string
	Package        string
	ControllerName string
	Router         string
	Methods        []*Router
}
type Router struct {
	Method   string
	Router   string
	FuncName string
}

func parsePkg(pkgRealpath, pkgpath string) ([]*Action, error) {
	var actions []*Action
	fileSet := token.NewFileSet()
	astPkgs, err := parser.ParseDir(fileSet, pkgRealpath, func(info os.FileInfo) bool {
		name := info.Name()
		return !info.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go")
	}, parser.ParseComments)

	if err != nil {
		return nil, err
	}
	for _, pkg := range astPkgs {
		for path, fl := range pkg.Files {
			action := &Action{}
			action, err = getAction(path)
			if err != nil {
				return nil, err
			}
			action.Path = pkgpath
			//获取action注解信息
			for _, d := range fl.Decls {
				switch specDecl := d.(type) {
				case *ast.FuncDecl:
					if specDecl.Recv != nil {
						exp, ok := specDecl.Recv.List[0].Type.(*ast.StarExpr) // Check that the type is correct first beforing throwing to parser
						if ok {
							if specDecl.Doc == nil {
								continue
							}
							//获取router注解
							router, err := getRouter(specDecl.Doc.List)
							if err != nil {
								return nil, err
							}
							if router == nil {
								continue
							}
							router.FuncName = specDecl.Name.String()
							//关联到action注解
							action.ControllerName = fmt.Sprint(exp.X)
							action.Methods = append(action.Methods, router)
						}
					}
				}
			}
			actions = append(actions, action)
		}
	}
	return actions, nil
}

func getRouter(lines []*ast.Comment) (*Router, error) {
	for _, c := range lines {
		router := &Router{}
		t := strings.TrimSpace(strings.TrimLeft(c.Text, "//"))
		if strings.HasPrefix(t, "@router") {
			t := strings.TrimSpace(strings.TrimLeft(c.Text, "//"))
			matches := routeRegex.FindStringSubmatch(t)
			if len(matches) == 3 {
				router.Router = matches[1]
				methods := matches[2]
				if methods == "" {
					router.Method = "get"
				} else {
					router.Method = methods
				}
				return router, nil
			} else {
				return nil, errors.New("Router information is missing")
			}
		}
	}
	return nil, nil
}
