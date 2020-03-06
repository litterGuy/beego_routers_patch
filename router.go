package routers

import (
	"errors"
	"fmt"
	"github.com/asktop/gotools/afile"
	"github.com/asktop/gotools/ajson"
	"github.com/asktop/gotools/akey"
	"github.com/wxnacy/wgo/arrays"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	//生成路由文件的前缀名称
	ROUTER_PREFIX         = "commentsRouter_"
	//项目url的统一前缀
	PROJECT_ROUTER_PREFIX = "/v1"
	//源码相对于根目录的存放位置
	PROJECT_SOURCE_CATALOG = "src"
)

func init() {
	err := generateRouter()
	if err != nil {
		panic(err)
	}
	fmt.Println("generate routers")
}

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
		body += "\tbeego.Router(\"" + router + "\"," + controllers + ",\"" + method.Method + ":" + method.FuncName + "\")\n";
	}
	body += "}"
	return body
}

//路由文件是否更改
func hasChanged(actions []*Action) bool {
	original := ajson.Encode(actions)
	fmt.Println(original)
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
	controllersPath := filepath.Join(projectPath, PROJECT_SOURCE_CATALOG, "controllers")
	if !afile.IsExist(controllersPath) {
		panic("controllers dir is not exists")
	}
	//获取project目录
	rootPath := filepath.Dir(projectPath)
	var actions []*Action
	err := filepath.Walk(controllersPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		//读取文件获取注解@action
		action, err := getAction(path)
		if err != nil {
			return err
		}
		if action == nil {
			return nil
		}
		//处理import地址
		path = strings.ReplaceAll(path, rootPath, "")
		path = strings.ReplaceAll(path, info.Name(), "")
		if strings.Contains(path, "\\") {
			path = strings.ReplaceAll(path, "\\", "/")
		}
		path = strings.Trim(path, "/")
		action.Path = path
		actions = append(actions, action)
		return nil
	})
	return actions, err
}

func getProjectPath() string {
	dir, _ := os.Getwd()
	return dir
}

//扫描单个文件
func getAction(path string) (*Action, error) {
	//增加标识，如果文件不含注解@action,跳过
	isAction := false
	//读取文件
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	action := &Action{}
	for i, line := range lines {
		//获取包名
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "package") {
			action.Package = strings.TrimSpace(strings.TrimLeft(t, "package"))
			continue
		}

		//获取action注解
		t = strings.TrimSpace(strings.TrimLeft(line, "//"))
		if strings.HasPrefix(t, "@action") {
			dataList := split(t)
			if len(dataList) != 2 && len(dataList) != 1 {
				return nil, errors.New("action annotate is error in file " + path)
			}
			if len(dataList) == 1 {
				action.Router = ""
			} else {
				action.Router = dataList[1]
			}

			isAction = true

			//获取controller名
			t = strings.TrimSpace(lines[i+1])
			if strings.HasPrefix(t, "type") {
				datas := split(t)
				if len(datas) != 4 {
					return nil, errors.New("action annotate is error in file " + path)
				}
				action.ControllerName = datas[1]
				continue
			}
		}

		//获取controller名
		if strings.HasPrefix(t, "type") {
			datas := split(t)
			if len(datas) != 4 {
				return nil, errors.New("action annotate is error in file " + path)
			}
			action.ControllerName = datas[1]
			continue
		}

		//获取router注解
		if strings.HasPrefix(t, "@router") {
			datas := split(t)
			datas = removeSpace(datas)
			if len(datas) != 3 {
				return nil, errors.New("router annotate is error in file " + path)
			}
			router := &Router{}
			router.Router = datas[1]
			method := datas[2]
			method = strings.ReplaceAll(method, "[", "")
			method = strings.ReplaceAll(method, "]", "")
			router.Method = method

			//获取methodName
			sl := lines[i+1]
			if strings.HasPrefix(sl, "func") {
				data := split(sl)
				data = removeSpace(data)
				if len(data) != 5 {
					return nil, errors.New("router annotate is error in file " + path)
				}
				funName := data[3]
				if strings.Contains(funName, "(") {
					funName = strings.ReplaceAll(funName, "(", "")
					funName = strings.ReplaceAll(funName, ")", "")
				}
				router.FuncName = funName
			}
			action.Methods = append(action.Methods, router)
		}
	}
	if isAction {
		return action, nil
	}
	return nil, nil
}

func removeSpace(str []string) []string {
	str = arrays.StringsDeduplicate(str)
	i := arrays.ContainsString(str, " ")
	if i > -1 {
		str = append(str[:i], str[i+1:]...)
	}
	k := arrays.ContainsString(str, "")
	if k > -1 {
		str = append(str[:k], str[k+1:]...)
	}
	return str
}

func split(str string) []string {
	arr := strings.Split(str, " ")
	if len(arr) < 2 {
		arr = strings.Split(str, "\t")
	} else {
		rst := []string{}
		for _, s := range arr {
			if strings.Contains(s, "\t") {
				tmpArr := strings.Split(s, "\t")
				rst = append(rst, tmpArr...)
			} else {
				rst = append(rst, s)
			}
		}
		arr = rst
	}
	return arr
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
