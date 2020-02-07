package new

func InfraInit() {
	fc1 := &FileContent{
		FileName: "infra.go",
		Dir:      "internal/infra",
		Content: `package infra

import (
	"sync"
	"github.com/google/wire"
	"github.com/jukylin/esim/container"
	"github.com/jukylin/esim/mysql"
	"{{PROPATH}}{{service_name}}/internal/infra/repo"
)

//Do not change the function name and var name
//  infraOnce
//  onceInfra
//  infraSet
//  NewInfra

var infraOnce sync.Once
var onceInfra *Infra

type Infra struct {
	*container.Esim

	DB *mysql.MysqlClient

	UserRepo repo.UserRepo
}

var infraSet = wire.NewSet(
	wire.Struct(new(Infra), "*"),
	provideEsim,
	provideDb,
	provideUserRepo,
)


func NewInfra() *Infra {
	infraOnce.Do(func() {
		onceInfra = initInfra()
	})

	return onceInfra
}

// Close close the infra when app stop
func (this *Infra) Close()  {

	this.DB.Close()
}

func (this *Infra) HealthCheck() []error {
	var errs []error
	var err error

	dbErrs := this.DB.Ping()
	if err != nil{
		errs = append(errs, dbErrs...)
	}

	return errs
}

func provideEsim() *container.Esim {


	return container.NewEsim()
}


func provideDb(esim *container.Esim) *mysql.MysqlClient {

	mysqlClientOptions := mysql.MysqlClientOptions{}
	mysqlClent := mysql.NewMysqlClient(
		mysqlClientOptions.WithConf(esim.Conf),
		mysqlClientOptions.WithLogger(esim.Logger),
	)

	return mysqlClent
}



func provideUserRepo(esim *container.Esim) repo.UserRepo {
	return repo.NewUserRepo(esim.Logger)
}
`,
	}

	fc2 := &FileContent{
		FileName: "wire.go",
		Dir:      "internal/infra",
		Content: `//+build wireinject

package infra

import (
	"github.com/google/wire"
)


func initInfra() *Infra {
	wire.Build(infraSet)
	return nil
}
`,
	}

	fc3 := &FileContent{
		FileName: "wire_gen.go",
		Dir:      "internal/infra",
		Content: `// Code generated by Wire. DO NOT EDIT.

//go:generate wire
//+build !wireinject

package infra

// Injectors from wire.go:

func initInfra() *Infra {
	esim := provideEsim()
	mysqlClient := provideDb(esim)
	userRepo := provideUserRepo(esim)
	infra := &Infra{
		Esim:     esim,
		DB:       mysqlClient,
		UserRepo: userRepo,
	}
	return infra
}
`,
	}

	Files = append(Files, fc1, fc2, fc3)
}
