/*
Copyright 2022 quarkcm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package osutil

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/CentaurusInfra/quarkcm/pkg/util/executil"
)

func Getenv(key string) string {
	return os.Getenv(key)
}

func Exists(name string) bool {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		return false
	}
	return true
}

func Mkdir(name string) {
	os.Mkdir(name, os.ModePerm)
}

func Create(fileName string) {
	os.Create(fileName)
}

func GetHostName() string {
	_, hostname, _ := executil.Execute("hostname")
	return strings.Replace(hostname, "\n", "", -1)
}

func ReadFromFile(fileName string) string {
	content, err := ioutil.ReadFile(fileName) // the file is inside the local directory
	if err != nil {
		return ""
	}
	return string(content)
}

func WriteToFile(fileName string, content string) {
	file, _ := os.Create(fileName)
	defer file.Close()
	file.WriteString(content)
}
