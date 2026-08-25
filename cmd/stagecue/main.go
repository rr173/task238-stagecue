package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"task238-stagecue/internal/httpapi"
	"task238-stagecue/internal/model"
	"task238-stagecue/internal/service"
	"task238-stagecue/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP 监听地址")
	dbPath := flag.String("db", "./stagecue.db", "SQLite 数据库路径")
	smoke := flag.Bool("smoke-test", false, "执行自检（真实落库+重启恢复）后退出，不启动长驻服务")
	flag.Parse()

	if *smoke {
		if err := runSmokeTest(*dbPath); err != nil {
			log.Fatalf("smoke-test failed: %v", err)
		}
		fmt.Println("smoke-test OK")
		return
	}

	st, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	svc := service.New(st)
	h := httpapi.NewHandler(svc)
	srv := &http.Server{Addr: *addr, Handler: h.Routes()}
	log.Printf("stagecue listening on %s (db=%s)", *addr, *dbPath)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server exit: %v", err)
	}
}

// runSmokeTest 真实写入演练、事件、提示、约束；校正时钟；构建时间线；检测冲突；
// 登记豁免；发布提示包；关闭并重新打开数据库验证重启恢复，最终以 0 退出。
func runSmokeTest(dbPath string) error {
	_ = os.Remove(dbPath)
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")

	st, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	svc := service.New(st)

	// 1. 导入演练版本
	rep, err := svc.Rehearsal.Create("smoke-show", "暗场撤离")
	if err != nil {
		return fmt.Errorf("create rehearsal: %w", err)
	}
	// 2. 设置时钟偏差（设备日志比脚本慢 50ms）
	if err := svc.Align.SetSkew(rep.ID, model.SourceScript, 0, true); err != nil {
		return err
	}
	if err := svc.Align.SetSkew(rep.ID, model.SourceCueLog, 0, false); err != nil {
		return err
	}
	if err := svc.Align.SetSkew(rep.ID, model.SourceDeviceLog, 50, false); err != nil {
		return err
	}
	// 3. 导入脚本事件（演员撤离安全区）t=1000
	if _, err := svc.Rehearsal.AddEvent(rep.ID, model.SourceScript, 0, model.ActorPerformer,
		model.RoleMoveOut, "actor-exit", 1000, ""); err != nil {
		return err
	}
	// 4. 导入灯光提示（暗场开始）t=900，结束 t=1100（设备日志 +50 => 950..1150，与演员撤离 1000 重叠）
	if _, err := svc.Rehearsal.AddEvent(rep.ID, model.SourceCueLog, 1, model.ActorPerformer,
		model.RoleCueStart, "blackout-start", 900, "L1"); err != nil {
		return err
	}
	if _, err := svc.Rehearsal.AddEvent(rep.ID, model.SourceCueLog, 2, model.ActorPerformer,
		model.RoleCueEnd, "blackout-end", 1100, "L1"); err != nil {
		return err
	}
	// 5. 导入设备响应（机械动作）t=980（设备日志 +50 => 1030，落入暗场区间）
	if _, err := svc.Rehearsal.AddEvent(rep.ID, model.SourceDeviceLog, 0, model.ActorPerformer,
		model.RoleMechMove, "trap-move", 980, "M1"); err != nil {
		return err
	}
	// 6. 新增约束：演员安全窗口内照明与机械最小间隔 100ms（容差）
	ct, err := svc.Constraint.Add(rep.ID, "performer-safety", model.ActorPerformer, 100, "演员撤离安全")
	if err != nil {
		return err
	}
	if err := svc.Constraint.Activate(ct.ID); err != nil {
		return err
	}
	// 7. 复核：应检测到冲突
	res, err := svc.RunReview(rep.ID)
	if err != nil {
		return err
	}
	if len(res.Conflicts) == 0 {
		return fmt.Errorf("expected at least one conflict, got 0")
	}
	// 8. 登记豁免（艺术需要：暗场掩护机械动作）
	conflictID := res.Conflicts[0].ID
	if _, err := svc.Waiver.Register(conflictID, "暗场掩护机械动作，演员已撤离", "blocking-plot-v2"); err != nil {
		return err
	}
	// 9. 再次复核，应无未解决冲突
	res2, err := svc.RunReview(rep.ID)
	if err != nil {
		return err
	}
	if res2.Unresolved != 0 {
		return fmt.Errorf("expected 0 unresolved after waiver, got %d", res2.Unresolved)
	}
	// 10. 发布提示包
	pkg, err := svc.CuePkg.Draft(rep.ID)
	if err != nil {
		return err
	}
	if _, err := svc.CuePkg.Publish(pkg.ID); err != nil {
		return err
	}

	// 重启恢复：关闭并重新打开数据库，验证数据仍在
	st.Close()
	st2, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("reopen: %w", err)
	}
	defer st2.Close()
	n, err := st2.Rehearsals().CountEvents(rep.ID)
	if err != nil {
		return err
	}
	if n != 4 {
		return fmt.Errorf("restart recovery: expected 4 events, got %d", n)
	}
	pkgs, err := st2.Packages().ListByRehearsal(rep.ID)
	if err != nil {
		return err
	}
	if len(pkgs) != 1 || pkgs[0].State != model.StatePublished {
		return fmt.Errorf("restart recovery: package not published/persisted")
	}
	return nil
}
