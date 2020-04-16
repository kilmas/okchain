package proxy

import (
	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/config"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/lite"
	lclient "github.com/tendermint/tendermint/lite/client"
	"github.com/tendermint/tendermint/types"
	dbm "github.com/tendermint/tm-db"
)

func NewVerifier(
	chainID,
	rootDir string,
	client lclient.SignStatusClient,
	logger log.Logger,
	cacheSize int,
) (*lite.DynamicVerifier, error) {

	logger = logger.With("module", "lite/proxy")
	logger.Info("lite/proxy/NewVerifier()...", "chainID", chainID, "rootDir", rootDir, "client", client)
	err := cmn.EnsureDir(rootDir, config.DefaultDirPerm)
	if err != nil {
		return nil, cmn.ErrorWrap(err, "ensure db path")
	}

	memProvider := lite.NewDBProvider("trusted.mem", dbm.NewMemDB()).SetLimit(cacheSize)
	lvlProvider := lite.NewDBProvider("trusted.lvl", dbm.NewDB("trust-base", dbm.GoLevelDBBackend, rootDir))
	trust := lite.NewMultiProvider(
		memProvider,
		lvlProvider,
	)
	source := lclient.NewProvider(chainID, client)
	cert := lite.NewDynamicVerifier(chainID, trust, source)
	cert.SetLogger(logger) // Sets logger recursively.

	// TODO: Make this more secure, e.g. make it interactive in the console?
	_, err = trust.LatestFullCommit(chainID, types.GetStartBlockHeight()+1, 1<<63-1)
	if err != nil {
		logger.Info("lite/proxy/NewVerifier found no trusted full commit, initializing from source from height 1...")
		fc, err := source.LatestFullCommit(chainID, types.GetStartBlockHeight()+1, types.GetStartBlockHeight()+1)
		if err != nil {
			return nil, errors.Wrap(err, "fetching source full commit @ height 1")
		}
		err = trust.SaveFullCommit(fc)
		if err != nil {
			return nil, errors.Wrap(err, "saving full commit to trusted")
		}
	}

	return cert, nil
}