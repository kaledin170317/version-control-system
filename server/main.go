package server

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"path/filepath"


	"github.com/nspcc-dev/neo-go/pkg/rpcclient"
	"github.com/nspcc-dev/neo-go/pkg/rpcclient/actor"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/spf13/viper"
)

type FileInfoPair struct {
  File string 
  Addr string 
}

func GetAllFiles(folderPath string) ([]string, error) {
  var files []string

  err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
    if err != nil {
      return err
    }


    if !info.IsDir() {
      files = append(files, path)
    }
    return nil
  })

  return files, err
}


func GetAllDirectories(folderPath string) ([]string, error) {
  var directories []string

  err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
    if err != nil {
      return err
    }


    if info.IsDir() {
      directories = append(directories, path)
    }
    return nil
  })

  return directories, err
}

func GetFileInfoPairs(ctx context.Context, s *Server, files []string) ([]FileInfoPair) {

  var filePairs []FileInfoPair
  for _, file := range files {
    addr, _ := putFile(ctx, s.p, s.acc, viper.GetString(cfgStorageContainer), file)
    filePairs = append(filePairs, FileInfoPair{
      File: file, 
      Addr: addr,     
    })
  }
  return filePairs
}

func RestoreFileInfoPairs(ctx context.Context, s *Server, files []FileInfoPair, dirFiles []string) {
  for _, dirFile := range dirFiles {
    err := os.Remove(dirFile)
    if err != nil {
      fmt.Println("Unable to delete file:", err)
    } else {
      fmt.Printf("Deleted file: %s\n", dirFile)
    }
	}

  for _, file := range files {
    getFile, _ := getFile(ctx, s.p, file.Addr)
    newPath := filepath.Join(filepath.Dir(file.File), filepath.Base(file.File))

    file, err := os.OpenFile(newPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
    if err != nil{
        fmt.Println("Unable to create file:", err) 
        os.Exit(1) 
    }

    _, _ = io.Copy(file, getFile.Payload)

    fmt.Println("Файл сохранён в %s",newPath)
  }
}


type VCS struct {
  Server Server
  Directory string
  PrevCommitHash string
}

func (vcs *VCS) CreateVCS(ctx context.Context, directory string) {
  server_, err  := NewServer(ctx)
  if (err != nil) {
    fmt.Println("Error: ", err)
  }
  vcs.Server = *server_
  vcs.Directory = directory
  vcs.PrevCommitHash = ""
}

func extractFilePaths(fileInfoPairs []FileInfoPair) ([]string, []string) {
  addrs := make([]string, len(fileInfoPairs))
  names := make([]string, len(fileInfoPairs))
  
  for i, pair := range fileInfoPairs {
    addrs[i] = pair.Addr
    names[i] = pair.File
  }
  
  return addrs, names
}

func createStringParamArray(parameters []string) smartcontract.Parameter{
  arrayParam := make([]smartcontract.Parameter, len(parameters))

  for i, _ := range arrayParam {
    arrayParam[i] = smartcontract.Parameter{
      Type: smartcontract.StringType,
      Value: parameters[i],
    }
  }

  return smartcontract.Parameter{
    Type: smartcontract.ArrayType,
    Value: arrayParam,
  }
}

func GenerateFrostFSCommitHash(fileIDs []string) string {
  sort.Strings(fileIDs)
  h := sha1.New()
  
  combinedIDs := strings.Join(fileIDs, "")
  
  h.Write([]byte(combinedIDs))
  
  hashBytes := h.Sum(nil)
  commitHash := hex.EncodeToString(hashBytes)
  
  return commitHash
}

func waitForTransaction(ctx context.Context, cli *rpcclient.Client, txID util.Uint256) error {
  ticker := time.NewTicker(time.Second * 5)
  defer ticker.Stop()

  for {
    select {
    case <-ctx.Done():
      return ctx.Err()

    case <-ticker.C:
      height, err := cli.GetTransactionHeight(txID)
      if err != nil {

        continue
      }
      if height > 0 {

        return nil
      }
    }
  }
}


type Commit struct { 
  Parent    string  
  FileInfoPairs     []FileInfoPair
}



func (vcs *VCS) FixDirState(ctx context.Context) string {
  files, _ := GetAllFiles(vcs.Directory)
  fileInfo := GetFileInfoPairs(ctx, &vcs.Server, files)

  fileAddreses, _ := extractFilePaths(fileInfo)

  fmt.Println("file addresses: ", fileAddreses)
  

  commitHash := "commit_" + GenerateFrostFSCommitHash(fileAddreses)

  commit := Commit{vcs.PrevCommitHash, fileInfo}

  a, _ := actor.NewSimple(vcs.Server.rpcCli, vcs.Server.acc)

  b := smartcontract.NewBuilder()

  commitMarshal, err := json.Marshal(commit)
  b.InvokeMethod(vcs.Server.contractHash, "putCommit", vcs.Server.acc.ScriptHash(), commitHash, commitMarshal)
  script, err := b.Script()
  if err != nil {
    fmt.Println("script error: ", err)
    return ""
  }

  res, err := a.Run(script)
  if res == nil {
    fmt.Println("res error: ", err)
    return ""
  }

  if res.State != "HALT" {
    panic("failed")
  }

  resVal, _ := res.Stack[0].TryBool()

  if !resVal {
    panic("transfer failed")
  }

  txID, _, err := a.SendRun(script)
  if err != nil {
    fmt.Printf("Ошибка отправки транзакции testPut: %v", err)
  }
  fmt.Printf("testPut TX отправлен: %s\n", txID.String())

  // Дожидаемся подтверждения транзакции да
  fmt.Println("Ждем, пока транзакция войдет в блок...")
  err = waitForTransaction(ctx, vcs.Server.rpcCli, txID)
  if err != nil {
    fmt.Printf("Ошибка ожидания транзакции: %v", err)
  }


  b.Reset()

  vcs.PrevCommitHash = commitHash

  return commitHash
}

func (vcs *VCS) GetDirState(ctx context.Context, commitHash string) bool {
  a, _ := actor.NewSimple(vcs.Server.rpcCli, vcs.Server.acc)
  b := smartcontract.NewBuilder()
  b.InvokeMethod(vcs.Server.contractHash, "getCommit", vcs.Server.acc.ScriptHash(), commitHash)
  script, err := b.Script()
  if err != nil {
    fmt.Println("script error: ", err)
    return false
  }

  res, err := a.Run(script)
  if res == nil {
    fmt.Println("res error: ", err)
    return false
  }
  if res.State != "HALT" {
    panic("failed")
  }

  resVal, _ := res.Stack[0].TryBytes()

  if resVal == nil {
    panic("transfer failed")
  }

  if err != nil {
    fmt.Println("errors of func invoke:", err)
    return false
  }

  var commit Commit
  err = json.Unmarshal(resVal, &commit)

  if err != nil {
		return false
	}

  fmt.Println(commit.FileInfoPairs)

  currDirFiles, _ := GetAllFiles(vcs.Directory)

  RestoreFileInfoPairs(ctx, &vcs.Server, commit.FileInfoPairs, currDirFiles)

  return true
}

func (vcs *VCS) GetHistory(ctx context.Context) string {
  a, _ := actor.NewSimple(vcs.Server.rpcCli, vcs.Server.acc)
  b := smartcontract.NewBuilder()
  b.InvokeMethod(vcs.Server.contractHash, "getHistory", vcs.Server.acc.ScriptHash())
  script, err := b.Script()
  if err != nil {
    fmt.Println("script error: ", err)
    return "Что-то пошло не так"
  }

  res, err := a.Run(script)
  if res == nil {
    fmt.Println("res error: ", err)
    return "Что-то пошло не так"
  }
  if res.State != "HALT" {
    panic("failed")
  }

  resVal, _ := res.Stack[0].TryBytes()
 
  if resVal == nil {
    return "Что-то пошло не так"
  }

  if err != nil {
    fmt.Println("errors of func invoke:", err)
    return "Что-то пошло не так"
  }


  if err != nil {
		return "Что-то пошло не так"
	}

  return string(resVal)
}