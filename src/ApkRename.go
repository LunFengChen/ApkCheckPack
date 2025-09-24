package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// APK信息结构体
type APKInfo struct {
	PackageName string
	VersionName string
	VersionCode string
	AppName     string
	FilePath    string
}

// 提取APK信息并打印
func ExtractAndPrintAPKInfo(apkPath string, apkReader *zip.Reader) *APKInfo {
	fmt.Println("\n===================== APK基本信息 =====================")
	
	info := &APKInfo{
		FilePath: apkPath,
	}

	// 优先使用zip解析AndroidManifest.xml（跨平台兼容）
	if apkReader != nil {
		parseManifestFromZip(apkReader, info)
		// 尝试从strings.xml获取应用名
		if info.AppName == "" {
			parseStringsFromZip(apkReader, info)
		}
	}

	// 如果zip解析失败，尝试使用aapt工具
	if info.PackageName == "" {
		if aaptInfo := extractWithAapt(apkPath); aaptInfo != nil {
			info.PackageName = aaptInfo.PackageName
			info.VersionName = aaptInfo.VersionName
			info.VersionCode = aaptInfo.VersionCode
			if info.AppName == "" {
				info.AppName = aaptInfo.AppName
			}
		}
	}

	// 如果应用名为空，使用文件名
	if info.AppName == "" {
		info.AppName = strings.TrimSuffix(filepath.Base(apkPath), ".apk")
	}

	// 清理字符串
	info.AppName = cleanString(info.AppName)
	info.PackageName = cleanString(info.PackageName)
	info.VersionName = cleanString(info.VersionName)

	// 打印APK信息
	fmt.Printf("文件路径: %s\n", apkPath)
	fmt.Printf("应用名称: %s\n", info.AppName)
	fmt.Printf("包名: %s\n", info.PackageName)
	fmt.Printf("版本名: %s\n", info.VersionName)
	fmt.Printf("版本号: %s\n", info.VersionCode)
	fmt.Println("========================================================")

	return info
}

// 使用aapt提取APK信息
func extractWithAapt(apkPath string) *APKInfo {
	cmd := exec.Command("aapt", "dump", "badging", apkPath)
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	info := &APKInfo{}
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if strings.HasPrefix(line, "package:") {
			if match := regexp.MustCompile(`name='([^']+)'`).FindStringSubmatch(line); len(match) > 1 {
				info.PackageName = match[1]
			}
			if match := regexp.MustCompile(`versionName='([^']+)'`).FindStringSubmatch(line); len(match) > 1 {
				info.VersionName = match[1]
			}
			if match := regexp.MustCompile(`versionCode='([^']+)'`).FindStringSubmatch(line); len(match) > 1 {
				info.VersionCode = match[1]
			}
		}
		
		if strings.HasPrefix(line, "application-label:") {
			if match := regexp.MustCompile(`application-label:'([^']+)'`).FindStringSubmatch(line); len(match) > 1 {
				info.AppName = match[1]
			}
		}
	}

	return info
}

// 从zip中解析AndroidManifest.xml
func parseManifestFromZip(apkReader *zip.Reader, info *APKInfo) {
	for _, file := range apkReader.File {
		if file.Name == "AndroidManifest.xml" {
			manifestReader, err := file.Open()
			if err != nil {
				continue
			}
			
			manifestData, err := io.ReadAll(manifestReader)
			manifestReader.Close()
			if err != nil {
				continue
			}

			manifestStr := string(manifestData)
			
			// 尝试多种模式匹配
			patterns := map[string]*regexp.Regexp{
				"package":     regexp.MustCompile(`package="([^"]+)"`),
				"versionName": regexp.MustCompile(`android:versionName="([^"]+)"`),
				"versionCode": regexp.MustCompile(`android:versionCode="([^"]+)"`),
				"label":       regexp.MustCompile(`android:label="([^"]+)"`),
			}
			
			for key, pattern := range patterns {
				if match := pattern.FindStringSubmatch(manifestStr); len(match) > 1 {
					switch key {
					case "package":
						info.PackageName = match[1]
					case "versionName":
						info.VersionName = match[1]
					case "versionCode":
						info.VersionCode = match[1]
					case "label":
						if info.AppName == "" && !strings.HasPrefix(match[1], "@") {
							info.AppName = match[1]
						}
					}
				}
			}
			break
		}
	}
}

// 从zip中解析strings.xml获取应用名
func parseStringsFromZip(apkReader *zip.Reader, info *APKInfo) {
	for _, file := range apkReader.File {
		// 查找strings.xml文件
		if strings.Contains(file.Name, "res/values") && strings.HasSuffix(file.Name, "strings.xml") {
			stringsReader, err := file.Open()
			if err != nil {
				continue
			}
			
			stringsData, err := io.ReadAll(stringsReader)
			stringsReader.Close()
			if err != nil {
				continue
			}

			stringsStr := string(stringsData)
			
			// 查找app_name
			patterns := []string{
				`<string name="app_name">([^<]+)</string>`,
				`<string name="app_title">([^<]+)</string>`,
				`<string name="application_name">([^<]+)</string>`,
				`<string name="title">([^<]+)</string>`,
			}
			
			for _, pattern := range patterns {
				if match := regexp.MustCompile(pattern).FindStringSubmatch(stringsStr); len(match) > 1 {
					info.AppName = strings.TrimSpace(match[1])
					return // 找到就返回
				}
			}
		}
	}
}

// 清理字符串，移除非法字符
func cleanString(s string) string {
	re := regexp.MustCompile(`[\/\\:*?"<>|]`)
	s = re.ReplaceAllString(s, "_")
	s = strings.Trim(s, " _")
	if s == "" {
		s = "Unknown"
	}
	return s
}

// 从加固检测结果中获取加固信息
func GetPackInfoFromResults() string {
	if len(allPackResults) == 0 {
		return "无加固"
	}
	
	// 从检测结果中提取加固厂商名称
	packNames := make(map[string]bool)
	for _, result := range allPackResults {
		// 解析结果格式: "    Sopath  梆梆安全（企业版） -> assets/libjiagu.so"
		parts := strings.Split(result, " -> ")
		if len(parts) >= 2 {
			// 提取加固厂商名称
			beforeArrow := strings.TrimSpace(parts[0])
			words := strings.Fields(beforeArrow)
			if len(words) >= 2 {
				packName := words[len(words)-1] // 取最后一个词作为加固厂商名
				
				// 清理加固名称，去掉括号内容
				packName = cleanPackName(packName)
				
				if packName != "" {
					packNames[packName] = true
				}
			}
		}
	}
	
	if len(packNames) == 0 {
		return "无加固"
	}
	
	// 将所有检测到的加固厂商用下划线连接
	var names []string
	for name := range packNames {
		names = append(names, name)
	}
	return strings.Join(names, "_")
}

// 清理加固名称，去掉括号内容和多余字符
func cleanPackName(packName string) string {
	// 去掉括号及其内容：梆梆安全（企业版） -> 梆梆安全
	re := regexp.MustCompile(`[（(][^）)]*[）)]`)
	packName = re.ReplaceAllString(packName, "")
	
	// 去掉一些常见的后缀词
	suffixes := []string{"加固", "保护", "安全", "防护"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(packName, suffix) && len(packName) > len(suffix) {
			// 只有当名称不只是后缀词时才去掉，比如"梆梆安全"保留，但"XXX安全"去掉"安全"
			continue
		}
	}
	
	// 清理空格和特殊字符
	packName = strings.TrimSpace(packName)
	packName = strings.ReplaceAll(packName, " ", "")
	
	return packName
}

// 重命名APK文件
func RenameAPKFile(apkInfo *APKInfo, packInfo string) error {
	if !*ArgRename {
		return nil
	}

	fmt.Println("\n===================== APK重命名操作 =====================")
	
	// 构建新文件名: <加固信息>_<appname>_<packagename>_<version>.apk
	var parts []string
	parts = append(parts, packInfo)
	parts = append(parts, apkInfo.AppName)
	parts = append(parts, apkInfo.PackageName)
	if apkInfo.VersionName != "" {
		parts = append(parts, apkInfo.VersionName)
	}
	
	newFileName := strings.Join(parts, "_") + ".apk"
	
	// 如果文件名太长，截断
	if len(newFileName) > 200 {
		baseName := strings.TrimSuffix(newFileName, ".apk")
		if len(baseName) > 196 {
			baseName = baseName[:196]
		}
		newFileName = baseName + ".apk"
	}
	
	// 获取目录和新文件路径
	dir := filepath.Dir(apkInfo.FilePath)
	newFilePath := filepath.Join(dir, newFileName)
	
	// 确保文件名唯一
	counter := 1
	originalNewFilePath := newFilePath
	for {
		if _, err := os.Stat(newFilePath); os.IsNotExist(err) {
			break
		}
		ext := filepath.Ext(originalNewFilePath)
		baseName := strings.TrimSuffix(originalNewFilePath, ext)
		newFilePath = fmt.Sprintf("%s_%d%s", baseName, counter, ext)
		counter++
	}
	
	fmt.Printf("原文件名: %s\n", filepath.Base(apkInfo.FilePath))
	fmt.Printf("新文件名: %s\n", filepath.Base(newFilePath))
	fmt.Printf("加固信息: %s\n", packInfo)
	
	// 复制文件
	err := copyFile(apkInfo.FilePath, newFilePath)
	if err != nil {
		return fmt.Errorf("复制文件失败: %v", err)
	}
	
	fmt.Printf("✓ APK重命名成功\n")
	
	// 如果需要删除原文件
	if *ArgDeleteOrig {
		err = os.Remove(apkInfo.FilePath)
		if err != nil {
			return fmt.Errorf("删除原文件失败: %v", err)
		}
		fmt.Printf("✓ 已删除原文件\n")
	}
	
	fmt.Println("========================================================")
	return nil
}

// 复制文件
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	return err
}
