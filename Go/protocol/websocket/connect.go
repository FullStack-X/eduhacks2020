package websocket

import (
	"eduhacks2020/Go/api"
	"eduhacks2020/Go/database"
	"eduhacks2020/Go/define/retcode"
	"eduhacks2020/Go/utils"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	//"go-websocket/api"
	//"go-websocket/define/retcode"
	//"go-websocket/tools/util"
	"net/http"
)

const (
	// 最大的消息大小
	maxMessageSize = 8192
)

// Controller 创建类型指定 run 方法
type Controller struct {
}

type renderData struct {
	ClientID string `json:"clientId"`
}

// Run 建立 Websocket 连接
func (c *Controller) Run(w http.ResponseWriter, r *http.Request, orm *database.ORM, r2 *database.RedisClient, m *database.MongoClientDevice, name string) {
	conn, err := (&websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// 允许所有CORS跨域请求
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}).Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("upgrade error: %v", err)
		//http.NotFound(w, r)
		return
	}

	//设置读取消息大小上线
	conn.SetReadLimit(maxMessageSize)

	//解析参数
	systemID := r.FormValue("systemId")
	if len(systemID) == 0 {
		_ = Render(conn, "", "", retcode.SystemIDError, "system ID cannot be empty", []string{})
		_ = conn.Close()
		return
	}

	clientID := utils.GenClientID()

	clientSocket := NewClient(clientID, systemID, conn, m)

	Manager.AddClient2SystemClient(systemID, clientSocket)

	store, dbSession := database.CreateMongoStore()
	defer dbSession.Close()
	session, err := store.Get(r, database.SessionName)

	if err != nil {
		log.Error(err.Error())
	}

	//读取客户端消息
	clientSocket.Read(orm, r2, session.ID, m, name)

	if err = api.ConnRender(conn, renderData{ClientID: clientID}); err != nil {
		_ = conn.Close()
		return
	}

	// 用户连接事件
	Manager.Connect <- clientSocket
}
