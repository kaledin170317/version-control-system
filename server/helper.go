package server

import (
	"context"
	"fmt"
	"os"
	"time"

	cid "git.frostfs.info/TrueCloudLab/frostfs-sdk-go/container/id"
	"git.frostfs.info/TrueCloudLab/frostfs-sdk-go/object"
	oid "git.frostfs.info/TrueCloudLab/frostfs-sdk-go/object/id"
	"git.frostfs.info/TrueCloudLab/frostfs-sdk-go/pool"
	"git.frostfs.info/TrueCloudLab/frostfs-sdk-go/user"
	"github.com/nspcc-dev/neo-go/pkg/rpcclient"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/nspcc-dev/neo-go/pkg/wallet"
	"github.com/spf13/viper"
)


const (
  cfgRPCEndpoint      = "rpc_endpoint"
  cfgRPCEndpointWS    = "rpc_endpoint_ws"
  cfgWallet           = "wallet"
  cfgPassword         = "password"
  cfgStorageNode      = "storage_node"
  cfgStorageContainer = "storage_container"
  cfgListenAddress    = "listen_address"
  cfgContractHash     = "contract_hash"
)

func putFile(ctx context.Context,p *pool.Pool, acc *wallet.Account, cnrIDStr string, filePath string) (string, error) {
  var cnrID cid.ID
  cnrID.DecodeString(cnrIDStr)

  var ownerID user.ID
  user.IDFromKey(&ownerID, acc.PrivateKey().PrivateKey.PublicKey)
  
  obj := object.New()
  obj.SetContainerID(cnrID)
  obj.SetOwnerID(ownerID)

  file, _ := os.Open(filePath)

  var prm2 pool.PrmObjectPut
  prm2.SetPayload(file)
  prm2.SetHeader(*obj)


  objID, _ := p.PutObject(ctx, prm2)
  return cnrID.EncodeToString() + "/" + objID.ObjectID.EncodeToString(), nil
}

func getFile(ctx context.Context,p *pool.Pool, addr string) (pool.ResGetObject, error ) {
  var getPrm pool.PrmObjectGet

  var address oid.Address
  address.DecodeString(addr)

  getPrm.SetAddress(address)

  res, _ := p.GetObject(ctx, getPrm)

  return res, nil
}

func createPool(ctx context.Context, acc *wallet.Account, addr string) (*pool.Pool, error) {
  var prm pool.InitParameters

  prm.SetKey(&acc.PrivateKey().PrivateKey)
  prm.AddNode(pool.NewNodeParam(1, addr, 1))
  prm.SetNodeDialTimeout(1 * time.Second)

  p, err := pool.NewPool(prm)
  if err != nil {
    return nil, fmt.Errorf("new Pool: %w", err)
  }

  err = p.Dial(ctx)
  if err != nil {
    return nil, fmt.Errorf("dial: %w", err)
  }

  return p, nil
}

type Server struct {
  p        *pool.Pool
  acc      *wallet.Account
  cnrID    cid.ID
  rpcCli    *rpcclient.Client
  contractHash util.Uint160
}

func NewServer(ctx context.Context) (*Server, error) {

	  viper.GetViper().SetConfigType("yml")
  	f, _ := os.Open("server/config.yml")
  	viper.GetViper().ReadConfig(f)

    fmt.Println(viper.GetString(cfgWallet))
    fmt.Println(viper.GetString(cfgPassword))
    fmt.Println(viper.GetString(cfgStorageNode))
    fmt.Println(viper.GetString(cfgStorageContainer))
  	w, err := wallet.NewWalletFromFile(viper.GetString(cfgWallet))
  	if err != nil {
    	return nil, err
  	}		

	acc := w.GetAccount(w.GetChangeAddress())
	if err = acc.Decrypt(viper.GetString(cfgPassword), w.Scrypt); err != nil {
		return nil, err
	}

	p, err := createPool(ctx, acc, viper.GetString(cfgStorageNode))
	if err != nil {
		return nil, err
	}


	var cnrID cid.ID
	if err = cnrID.DecodeString(viper.GetString(cfgStorageContainer)); err != nil {
		return nil, err
	}

	rpcClient, err := rpcclient.New(ctx, viper.GetString(cfgRPCEndpoint), rpcclient.Options{})
  	if err != nil {
    	return nil, fmt.Errorf("failed to create Neo client: %w", err)
  	}

	contractHashFromCfg := viper.GetString(cfgContractHash)
	decodedContractHash, err := util.Uint160DecodeStringLE(contractHashFromCfg)
	if err != nil {
    	return nil, fmt.Errorf("failed to create Neo client: %w", err)
  	}

	return &Server{
		p:        p,
		acc:      acc,
		cnrID:    cnrID,
		rpcCli: rpcClient,
		contractHash: decodedContractHash,
	}, nil
}