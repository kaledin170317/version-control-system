package contract

import (
	"github.com/nspcc-dev/neo-go/pkg/interop/runtime"
	"github.com/nspcc-dev/neo-go/pkg/interop/storage"
	"github.com/nspcc-dev/neo-go/pkg/interop"
)


type FileInfoPair struct {
  File string 
  Addr string 
}

type Commit struct {       
  Hash      string  
  Parent    string  
  FileInfoPairs     []FileInfoPair
}

const (
  	ownerKey = "owner"
    historyKey = "history"
)

func _deploy(data interface{}, isUpdate bool) {
  if !isUpdate {
    ctx := storage.GetContext()
    owner := runtime.GetExecutingScriptHash()
    storage.Put(ctx, ownerKey, owner)
    history := []byte("И Востали машины из пепла ядерного огня\n")
    storage.Put(ctx, historyKey, history)
    runtime.Log("Contract deployed and owner set")
  }

}

// PutCommit сохраняет структуру Commit в хранилище контракта.
func PutCommit(author interop.Hash160, commitHash string, com []byte) bool {
  if !runtime.CheckWitness(author) {
    runtime.Log("Unauthorized access")
    return false
  }
  
  ctx := storage.GetContext()
  storage.Put(ctx, commitHash, com)


  history := storage.Get(ctx, historyKey).(string)
  uphistory := history + commitHash + "\n"
  storage.Put(ctx, historyKey, uphistory)



  checkData := storage.Get(ctx, commitHash)
  if checkData == nil {
      runtime.Log("Failed to save commit")
      return false
  }
  
  runtime.Log("Commit saved successfully")
  runtime.Log(commitHash)

  return true
}

// GetCommit получает структуру Commit из хранилища контракта.
func GetCommit(author interop.Hash160, commitHash string) []byte {
  runtime.Log(commitHash)
  ctx := storage.GetContext()

  if !runtime.CheckWitness(author) {
        runtime.Log("Учи уроки")
    return []byte{}
  }

  data := storage.Get(ctx, commitHash)



  if data == nil {
    runtime.Log("Пусто на душе")
  }

  decodedCommit := data.([]byte)

  return decodedCommit
}

func GetHistory(author interop.Hash160) []byte {

  if !runtime.CheckWitness(author) {
    runtime.Log("Учи уроки")
    return nil
  }
 
  ctx := storage.GetContext()
  history := storage.Get(ctx, historyKey)

  if history == nil {
      runtime.Log("No history found")
      return nil
  }

  return history.([]byte)
}
