package qconfig

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/kamioair/qf/utils/qio"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"strings"
)

var (
	_filePath = "./config/config.yaml"
)

// 默认配置文件內容
//
//go:embed config.yaml
var configContent []byte

// ChangeFilePath
//
//	@Description: 修改文件路径
//	@param module 新文件路径
func ChangeFilePath(path string) {
	_filePath = path
	// 没有默认值则不生成文件
	if configContent != nil && len(configContent) > 0 {
		if qio.PathExists(_filePath) == false {
			err := qio.WriteAllBytes(_filePath, configContent, false)
			if err != nil {
				panic(fmt.Errorf("Fatal error config file: %s \n", err))
			}
		}
	}
	viper.SetConfigFile(_filePath)
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
}

// Load
//
//	@Description: 加载内容到结构体
//	@param module 模块名称
//	@param configStruct 结构体指针
func Load(module string, configStruct any) {
	// 读取文件
	value := viper.Get(module)
	if value == nil {
		return
	}
	js, err := json.Marshal(value)
	if err == nil {
		err = json.Unmarshal(js, configStruct)
		if err != nil {
			return
		}
	}
}

// Get
//
//	@Description: 获取内容
//	@param module 模块名称
//	@param key 节点路径，例如 xx.xx.xx
//	@param defValue 默认值
//	@return T
func Get[T any](module, key string, defValue T) T {
	// 从缓存中获取
	tKey := fmt.Sprintf("%s.%s", module, key)
	if module == "" {
		tKey = key
	}
	value := viper.Get(tKey)
	if value == nil && module != "" {
		// 尝试找默认配置
		value = viper.Get(key)
	}

	// 写入默认值
	if value == nil {
		viper.SetDefault(tKey, defValue)
		value = defValue
	}

	// 转换具体指
	js, err := json.Marshal(value)
	if err == nil {
		newObj := new(T)
		err := json.Unmarshal(js, &newObj)
		cont := string(js)
		cont = cont + ""
		if err == nil {
			return *newObj
		}
	}
	return *new(T)
}

// Save
//
//	@Description: 保持结构体的值
//	              注意，当前版本该方法只会替换配置文件中已经存在的节点的值，
//				  新增的属性不会自动创建
//	@param module 模块名称
//	@param configStruct 结构体指针
func Save(module string, configStruct any) error {
	yamlData, err := qio.ReadAllBytes(_filePath)
	if err != nil {
		return err
	}
	// 更新YAML数据
	var model any
	if module == "" {
		model = configStruct
	} else {
		model = map[string]interface{}{module: configStruct}
	}
	jsonStr, err := json.Marshal(model)
	if err != nil {
		return err
	}
	updatedYAML, err := updateYAMLWithJSON(yamlData, string(jsonStr))
	if err != nil {
		return err
	}
	// 过滤多余空格
	str := string(updatedYAML)
	str = strings.Replace(str, "\n\n", "\n", -1)
	sp := strings.Split(str, "\n")
	final := ""
	for i, line := range sp {
		if i > 0 && strings.HasPrefix(line, "#") && !strings.HasPrefix(sp[i-1], "#") {
			final += "\n"
		}
		if !strings.HasPrefix(line, "#") {
			line = strings.Replace(line, "    ", "  ", -1)
		}
		final += fmt.Sprintf("%s\n", line)
	}
	// 写回文件
	err = qio.WriteAllBytes(_filePath, []byte(final), false)
	if err != nil {
		return err
	}
	return nil
}

// updateNodeValue 更新yaml.Node中的值
func updateNodeValue(node *yaml.Node, key string, value interface{}) {
	if node == nil || node.Content == nil {
		return
	}
	for i, n := range node.Content {
		if n.Kind == yaml.ScalarNode && n.Value == key {
			// 找到对应的键，更新其值
			node.Content[i+1].Value = fmt.Sprintf("%v", value)
			return
		}
	}
}

// updateYAMLWithJSON 将JSON字符串转换为interface{}，并更新YAML节点
func updateYAMLWithJSON(yamlData []byte, jsonStr string) ([]byte, error) {
	// 解析JSON字符串
	var jsonData interface{}
	err := json.Unmarshal([]byte(jsonStr), &jsonData)
	if err != nil {
		return nil, err
	}

	// 解析YAML文件
	var root yaml.Node
	err = yaml.Unmarshal(yamlData, &root)
	if err != nil {
		return nil, err
	}

	// 递归更新YAML节点
	updateYAMLNode(&root, jsonData, []string{})

	// 将更新后的YAML节点转换为字节数组
	return yaml.Marshal(&root)
}

// updateYAMLNode 递归更新YAML节点
func updateYAMLNode(node *yaml.Node, data interface{}, keys []string) {
	if node == nil {
		return
	}
	switch node.Kind {
	case yaml.DocumentNode:
		updateYAMLNode(node.Content[0], data, keys)
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]
			newKeys := append(keys, keyNode.Value)
			value := findValueInData(data, newKeys)
			if value != nil {
				if valueNode.Tag != "!!map" {
					updateNodeValue(node, keyNode.Value, value)
				}
			}
			updateYAMLNode(valueNode, data, newKeys)
		}
	case yaml.SequenceNode:
		for i, n := range node.Content {
			updateYAMLNode(n, data, append(keys, fmt.Sprintf("%d", i)))
		}
	}
}

// findValueInData 在data中查找指定键的值
func findValueInData(data interface{}, keys []string) interface{} {
	if len(keys) == 0 {
		return nil
	}
	switch tp := data.(type) {
	case map[string]interface{}:
		nv := map[string]interface{}{}
		for k, v := range tp {
			nv[strings.ToLower(k)] = v
		}
		if value, ok := nv[strings.ToLower(keys[0])]; ok {
			if len(keys) == 1 {
				return value
			}
			return findValueInData(value, keys[1:])
		}
	case []interface{}:
		for _, item := range tp {
			if value := findValueInData(item, keys); value != nil {
				return value
			}
		}
	}
	return nil
}
