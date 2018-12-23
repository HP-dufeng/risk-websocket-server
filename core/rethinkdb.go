package core

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

var (
	DbName                             = "rma_subscribes"
	TableName_SubscribeTunnelRealFund  = "SubscribeTunnelRealFund"
	TableName_SubscribeCorpHoldMon     = "SubscribeCorpHoldMon"
	TableName_SubscribeQuoteMon        = "SubscribeQuoteMon"
	TableName_SubscribeCustRisk        = "SubscribeCustRisk"
	TableName_SubscribeCustHold        = "SubscribeCustHold"
	TableName_SubscribeCustGroupHold   = "SubscribeCustGroupHold"
	TableName_SubscribeProuctGroupRisk = "SubscribeProuctGroupRisk"
	TableName_SubscribeNearDediveHold  = "SubscribeNearDediveHold"
)

func Subscribe(session *r.Session, tableName string) {
	res, err := r.Table(tableName).Changes().Run(session)
	if err != nil {
		log.Fatalln(err)
	}

	var value interface{}

	log.Infof("%s received start", tableName)
	for res.Next(&value) {
		log.Info(value)
	}

}

func checkIfRethinkDBReady(ctx context.Context, rethinkAddr string) *r.Session {
	log.Infoln("Check if rethinkdb ready ...")
	defer log.Infoln("Check if rethinkdb ready ... ok")

	waitForDBReady := make(chan *r.Session)

	go func() {
		select {
		default:
			for {
				session, err := r.Connect(r.ConnectOpts{
					Address: rethinkAddr, // endpoint without http
				})
				if err != nil {
					log.Errorln(err)
					time.Sleep(5 * time.Second)
					continue
				}
				log.Info("RethinkDB connected successed")

				waitForDBReady <- session
				return
			}
		case <-ctx.Done():
			waitForDBReady <- nil
			return
		}
	}()

	session := <-waitForDBReady

	return session
}
