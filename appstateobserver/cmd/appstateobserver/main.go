package main

import (
	"context"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/appstate"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/appstateobserver"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/appstateobserver/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/appstateobserver/server"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/auth"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/startup"
	"github.com/tedsuo/ifrit"
)

func main() {
	conf, logger := startup.Bootstrap("appstateobserver", config.LoadConfig)
	policyDB := startup.CreatePolicyDB(conf.Db[db.PolicyDb], logger)
	defer func() { _ = policyDB.Closer() }()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	store, err := appstateobserver.NewStore(ctx, conf.Db[db.ScalingEngineDb])
	cancel()
	startup.ExitOnError(err, logger, "connect observer database")
	defer func() { _ = store.Close() }()
	cfClient, err := cf.NewCFClientWrapper(&conf.CF, logger)
	startup.ExitOnError(err, logger, "create observer CF client")
	observer := &appstateobserver.Observer{Store: store, CF: cfClient, Membership: policyDB.DB,
		Logger: logger, ReconcileInterval: conf.ReconcileInterval, Window: conf.Window}
	srv := server.NewServer(logger, conf, store)
	am := auth.NewXfccAuthMiddleware(logger, conf.CFServer.XFCC)
	startup.StartService(logger,
		startup.Server("audit-reader", func() (ifrit.Runner, error) {
			return appstate.Runner(logger.Session("audit-reader"), conf.PollInterval, observer.Poll), nil
		}),
		startup.Server("state-refresh", func() (ifrit.Runner, error) {
			return appstate.Runner(logger.Session("state-refresh"), time.Second, observer.Refresh), nil
		}),
		startup.Server("cf-server", func() (ifrit.Runner, error) { return srv.CreateCFServer(am) }),
		startup.Server("health-server", srv.CreateHealthServer),
	)
}
