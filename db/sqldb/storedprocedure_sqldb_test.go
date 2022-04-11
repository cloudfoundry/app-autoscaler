package sqldb_test

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db/sqldb"
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const instanceId = "InstanceId1"
const bindingId = "BindingId1"

var _ = Describe("Stored Procedure test", func() {
	var (
		storedProcedure *sqldb.StoredProcedureSQLDb
		dbConfig        db.DatabaseConfig
		logger          lager.Logger
		err             error
	)

	BeforeEach(func() {
		logger = lager.NewLogger("stored_procedure")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
		dbConfig = db.DatabaseConfig{
			URL:                   os.Getenv("DBURL"),
			MaxOpenConnections:    10,
			MaxIdleConnections:    5,
			ConnectionMaxLifetime: 10 * time.Second,
			ConnectionMaxIdleTime: 10 * time.Second,
		}

		if !strings.Contains(os.Getenv("DBURL"), "postgres") {
			Skip("Not configured for Postgres")
		}

		if getPostgresMajorVersion() < 12 {
			Skip("This test only works for Postgres v12 and above")
		}

		deleteAllFunctions()
		addPSQLFunctions()
	})

	AfterEach(func() {
		deleteAllFunctions()
	})

	Describe("NewBindingSQLDB", func() {
		JustBeforeEach(func() {
			storedProcedure, err = sqldb.NewStoredProcedureSQLDb(models.StoredProcedureConfig{
				SchemaName:                             "public",
				CreateBindingCredentialProcedureName:   "create_creds",
				DropBindingCredentialProcedureName:     "deleteCreds",
				DropAllBindingCredentialProcedureName:  "deleteAll",
				ValidateBindingCredentialProcedureName: "validate",
			}, dbConfig, logger)
		})

		AfterEach(func() {
			if storedProcedure != nil {
				err = storedProcedure.Close()
				Expect(err).NotTo(HaveOccurred())
			}
		})

		When("create is called", func() {
			It("it returns the bindingId and instanceId as result", func() {
				creds, err := storedProcedure.CreateCredentials(models.CredentialsOptions{
					InstanceId: instanceId,
					BindingId:  bindingId,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(creds.Username).To(Equal("BindingId1 from create"))
				Expect(creds.Password).To(Equal("InstanceId1 from create"))
			})
		})
		When("delete is called", func() {
			It("is successful", func() {
				err := storedProcedure.DeleteCredentials(models.CredentialsOptions{
					InstanceId: instanceId,
					BindingId:  bindingId,
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})
		When("DeleteAllInstanceCredentials is called", func() {
			It("is successful", func() {
				err := storedProcedure.DeleteAllInstanceCredentials(instanceId)
				Expect(err).NotTo(HaveOccurred())
			})
		})
		When("DeleteAllInstanceCredentials is called", func() {
			It("is successful", func() {
				credOpts, err := storedProcedure.ValidateCredentials(models.Credential{
					Username: instanceId,
					Password: bindingId,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(credOpts.InstanceId).To(Equal("InstanceId1 from validate"))
				Expect(credOpts.BindingId).To(Equal("BindingId1 from validate"))
			})
		})
	})
})

func getPostgresMajorVersion() int {
	var version string
	err := dbHelper.QueryRow("select version()").Scan(&version)
	if err != nil {
		Fail(fmt.Sprintf("postgres db failure while getting version:%s", err.Error()))
	}
	r, _ := regexp.Compile("PostgreSQL ([0-9]+)")
	versionNumber, err := strconv.Atoi(r.FindStringSubmatch(version)[1])
	if err != nil {
		Fail(fmt.Sprintf("Could not determine postgress version from string: '%s'", version))
	}
	return versionNumber
}

func deleteAllFunctions() {
	deleteFunction("deleteCreds")
	deleteFunction("deleteAll")
	deleteFunction("validate")
	deleteFunction("create_creds")
}

func addPSQLFunctions() {
	addCreateFunction()
	addDeleteFunction()
	addDeleteAllFunction()
	addValidateFunction()
}

func addCreateFunction() {
	//nolint:rowserrcheck
	_, err := dbHelper.Query(`
create or replace function create_creds(
  username varchar,
  password varchar
) returns TABLE(username varchar, password varchar)
language SQL
as $$ SELECT $2 || ' from create', $1 || ' from create' $$`)
	if err != nil {
		Fail(fmt.Sprintf("could not create function createCreds: %s", err.Error()))
	}
}

func addDeleteFunction() {
	//nolint:rowserrcheck
	_, err := dbHelper.Query(fmt.Sprintf(`
create or replace function "deleteCreds"(
  username varchar,
  password varchar
) returns integer
language plpgsql
as $$
begin
    if username != '%s' or password != '%s' then
        RAISE unique_violation USING MESSAGE = 'invalid password and username combination';
    end if;
    return 1;
end;
$$`, instanceId, bindingId))
	if err != nil {
		Fail(fmt.Sprintf("could not create function deleteCreds: %s", err.Error()))
	}
}

func addDeleteAllFunction() {
	//nolint:rowserrcheck
	_, err := dbHelper.Query(fmt.Sprintf(`
create or replace function "deleteAll"( instanceId varchar) 
returns integer
language plpgsql
as $$
begin
    if instanceId != '%s' then
         RAISE EXCEPTION 'invalid instanceId %%',instanceId ;
    end if;
    return 1;
end;
$$`, instanceId))
	if err != nil {
		Fail(fmt.Sprintf("could not create function deleteAll: %s", err.Error()))
	}
}

func addValidateFunction() {
	//nolint:rowserrcheck
	_, err := dbHelper.Query(fmt.Sprintf(`
create or replace function "validate"( username text, password text) 
returns TABLE( instanceId text, bindingId text)
language plpgsql
as $$
begin
    if username != '%s' or password != '%s' then
         RAISE EXCEPTION 'invalid username and password' ;
    end if;
    return query SELECT username || ' from validate' , password || ' from validate'  ;
end;
$$`, instanceId, bindingId))
	if err != nil {
		Fail(fmt.Sprintf("could not create function validate: %s", err.Error()))
	}
}

func deleteFunction(name string) {
	//nolint:rowserrcheck
	_, err := dbHelper.Query(fmt.Sprintf("Drop function if exists public.%s", pq.QuoteIdentifier(name)))
	if err != nil {
		Fail(fmt.Sprintf("could not remove procedure %s: %s", name, err.Error()))
	}
}
