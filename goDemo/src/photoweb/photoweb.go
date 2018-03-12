package main 

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"runtime/debug"
	"strings"
)

var PWD, _ = os.Getwd()
var (
	// Question: 只能使用绝对路径，如何使用相对路径
	UPLOAD_DIR = PWD + "/src/photoweb/uploads"
	TEMPLATE_DIR = PWD + "/src/photoweb/views"
)

var templates = make(map[string]*template.Template)

func init() {
	// log.Println(TEMPLATE_DIR)
	fileInfoArr, err := ioutil.ReadDir(TEMPLATE_DIR)
	check(err)
	var templateName, templatePath string
	for _, fileInfo := range fileInfoArr {
		templateName = fileInfo.Name()
		if ext := path.Ext(templateName); ext != ".html" {
			continue
		}
		templatePath = TEMPLATE_DIR+"/"+templateName
		log.Println("Loading template:", templatePath)
		// template.Must() 确保模板不能解析成功时，一定会触发错误处理流程
		// 如果模板不能成功加载，程序能做的唯一有意义的事情就是退出
		t := template.Must(template.ParseFiles(templatePath))
		tmpl := strings.Split(templateName, ".")[0]
		templates[tmpl] = t
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		log.Println("Got Get Request:", r)
		if err := renderHtml(w, "upload", nil); err!= nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	case "POST":
		log.Println("Got Post Request:", r)
		f, h, err := r.FormFile("image")
		if err != nil {
			http.Error(w, err.Error(),
			http.StatusInternalServerError)
			log.Println("上传接收失败")
			return
		}
		filename := h.Filename
		defer f.Close()
		t, err := os.Create(UPLOAD_DIR+"/"+filename)
		if err != nil {
			http.Error(w, err.Error(),
				http.StatusInternalServerError)
			log.Println("临时文件创建失败")
			return
		}
		defer t.Close()
		if _, err := io.Copy(t, f); err != nil {
			http.Error(w, err.Error(),
				http.StatusInternalServerError)
			log.Print("图片副本保存失败")
			return
		}
		http.Redirect(w, r, "/view?id="+filename,
			http.StatusFound)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	imageId := r.FormValue("id")
	imagePath := UPLOAD_DIR + "/" + imageId
	if exists := isExists(imagePath); !exists {
		http.NotFound(w, r)
		return
	}
	// 设置显示格式
	w.Header().Set("Content-type", "image")
	// ServeFile() 将imagePath路径下的文件从磁盘中读取并作为服务端的返回信息输出给客户端
	http.ServeFile(w, r, imagePath)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func isExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return os.IsExist(err)
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	fileInfoArr, err := ioutil.ReadDir(UPLOAD_DIR)
	check(err)
	locals := make(map[string]interface{})
	images := []string{}
	for _, fileInfo := range fileInfoArr {
		images = append(images, fileInfo.Name())
	}
	locals["images"] = images
	if err := renderHtml(w, "list", locals); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func renderHtml(w http.ResponseWriter, tmpl string, locals map[string]interface{}) (err error) {
	// template.ParseFiles() 函数将会读取指定模板的内容并且返回一个*template.Template值
	// t.Execute() 根据模板语法来执行模板的渲染，并将渲染后的结果作为HTTP的返回数据输出
	err = templates[tmpl].Execute(w, locals)
	return err
}


// 使用闭包
// 同时使用defer和recover()方法终结panic
// 封住了业务逻辑处理函数，使得更加程序更加健壮
func safeHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err, ok := recover().(error); ok {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				log.Printf("WARN: panic in %v -%v", fn, err)
				log.Println(string(debug.Stack()))
			}
		}()
		fn(w, r)
	}
}


func main() {
	// HandleFunc类似路由
	http.HandleFunc("/", safeHandler(listHandler))
	http.HandleFunc("/view", safeHandler(viewHandler))
	http.HandleFunc("/upload", safeHandler(uploadHandler))
	// 开启服务并进行监听
	fmt.Println("HTTP Server listen and serve at 0.0.0.0:8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}

