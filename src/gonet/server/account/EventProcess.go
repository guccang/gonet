package account

import (
	"database/sql"
	"fmt"
	"gonet/actor"
	"gonet/base"
	"gonet/db"
	"gonet/message"
	"log"
)

type (
	EventProcess struct {
		actor.Actor
		m_db *sql.DB
	}

	IEventProcess interface {
		actor.IActor
	}
)

func (this *EventProcess) Init(num int) {
	this.Actor.Init(num)
	this.m_db = SERVER.GetDB()
	//创建账号
	this.RegisterCall("C_A_RegisterRequest", func(packet *message.C_A_RegisterRequest) {
		accountName := packet.GetAccountName()
		//password := *packet.Password
		password := "123456"
		socketId := int(this.GetRpcPacket().RpcHead.Id)
		nError := 1
		accountId := base.UUID.UUID()
		//查找账号存在
		rows, err := this.m_db.Query(fmt.Sprintf("select 1 from tbl_account A where A.account_name = '%s'", accountName))
		if err == nil{
			rs := db.Query(rows, err)
			if !rs.Next(){
				//创建账号
				_, err := this.m_db.Exec(fmt.Sprintf("insert into tbl_account (account_name, password, account_id) values('%s', '%s', %d)", accountName, base.MD5(password), accountId))
				if (err == nil) {
					SERVER.GetLog().Printf("帐号[%s]创建成功", accountName)
					//登录账号
					SERVER.GetAccountMgr().SendMsg( "Account_Login", accountName, accountId, socketId, this.GetSocketId())
					nError = 0
				}
			}else{//账号存在
				SERVER.GetLog().Printf("帐号[%s]已存在", accountName)
			}
		}

		if nError != 0 {
			SendToClient(this.GetSocketId(), &message.A_C_RegisterResponse{
				PacketHead: message.BuildPacketHead( accountId, 0),
				Error:      int32(nError),
			}, &message.RpcHead{Id:this.GetRpcPacket().RpcHead.Id})
		}
	})

	//登录账号
	this.RegisterCall("C_A_LoginRequest", func(packet *message.C_A_LoginRequest) {
		accountName := packet.GetAccountName()
		//password := *packet.Password
		password := "123456"
		buildVersion := packet.GetBuildNo()
		socketId := int(this.GetRpcPacket().RpcHead.Id)
		nError := base.NONE_ERROR

		if accountName == ""{
			nError = base.ACCOUNT_NOEXIST
		} else if base.VERSION.IsAcceptableBuildVersion(buildVersion) {
			log.Printf("账号[%s]登陆账号服务器", accountName)
			rows, err := this.m_db.Query(fmt.Sprintf("select account_id, password from tbl_account where account_name = '%s'", accountName))
			if err == nil {
				rs := db.Query(rows, err)
				if rs.Next(){
					accountId := rs.Row().Int64("account_id")
					passWd := rs.Row().String("password")
					if base.MD5(password)== passWd{
						nError = base.NONE_ERROR
						SERVER.GetAccountMgr().SendMsg("Account_Login", accountName, accountId, socketId, this.GetSocketId())
					}else{//密码错误
						nError = base.PASSWORD_ERROR
					}
				}else{
					nError = base.ACCOUNT_NOEXIST
				}
			}
		} else {
			nError = base.VERSION_ERROR
			log.Printf("版本验证错误 clientVersion=%s,err=%d", buildVersion, nError)
		}

		if nError != base.NONE_ERROR {
			SendToClient(this.GetSocketId(), &message.A_C_LoginResponse{
				PacketHead:message.BuildPacketHead( 0, 0 ),
				Error:int32(nError),
				AccountName:packet.AccountName,
			}, &message.RpcHead{Id:this.GetRpcPacket().RpcHead.Id})
		}
	})

	//创建玩家
	this.RegisterCall("W_A_CreatePlayer", func(accountId int64, playername string, sex int32, socketId int) {
		playerId := base.UUID.UUID()
		_, err := this.m_db.Exec(fmt.Sprintf("insert into tbl_player (account_id, player_name, player_id) values (%d, '%s', %d)", accountId, playername, playerId))
		if err == nil {
			SERVER.GetServer().SendMsgById(this.GetSocketId(), "A_W_CreatePlayer", accountId, playerId, playername, sex, socketId)
		}
	})

	//删除玩家
	this.RegisterCall("W_A_DeletePlayer", func(accountId int64, playerId int64) {
		this.m_db.Exec(fmt.Sprintf("update tbl_player set delete_flag = 1 where account_id =%d and player_id=%d", accountId, playerId))
	})

	this.Actor.Start()
}
