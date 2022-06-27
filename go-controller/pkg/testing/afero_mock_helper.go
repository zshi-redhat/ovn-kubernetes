package testing

import "os"

type AferoFileMockHelper struct {
	FileName    string
	Content     []byte
	Permissions os.FileMode
}

type AferoDirMockHelper struct {
	DirName     string
	Files       []AferoFileMockHelper
	Permissions os.FileMode
}
