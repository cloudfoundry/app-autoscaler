package sqldb_test

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db/sqldb"
	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var instanceId = "InstanceId1"
var bindingId = "BindingId1"

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
		dbUrl := testhelpers.GetDbUrl()
		dbConfig = db.DatabaseConfig{
			URL:                   dbUrl,
			MaxOpenConnections:    10,
			MaxIdleConnections:    5,
			ConnectionMaxLifetime: 10 * time.Second,
			ConnectionMaxIdleTime: 10 * time.Second,
		}

		if !strings.Contains(dbUrl, "postgres") {
			Skip("Postgres test")
		}

		if getPostgresMajorVersion() < 12 {
			Skip("This test only works for Postgres v12 and above")
		}
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
				creds, err := storedProcedure.CreateCredentials(context.Background(), models.CredentialsOptions{
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
				err := storedProcedure.DeleteCredentials(context.Background(), models.CredentialsOptions{
					InstanceId: instanceId,
					BindingId:  bindingId,
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})
		When("DeleteAllInstanceCredentials is called", func() {
			It("is successful", func() {
				err := storedProcedure.DeleteAllInstanceCredentials(context.Background(), instanceId)
				Expect(err).NotTo(HaveOccurred())
			})
		})
		When("ValidateCredentials", func() {
			It("is successful", func() {
				// The binding_id as returned from the stored-procedure is – for testing-purposes –
				// slightly modified.
				bindingIdFromStoredProd := fmt.Sprintf("%s from validate", bindingId)
				credOpts, err := storedProcedure.ValidateCredentials(context.Background(), models.Credential{
					Username: instanceId,
					Password: bindingId,
				}, bindingIdFromStoredProd)
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
	rows, err := dbHelper.Query(`
create or replace function create_creds(
  username varchar,
  password varchar
) returns TABLE(username varchar, password varchar)
language SQL
as $$ SELECT $2 || ' from create', $1 || ' from create' $$`)
	defer func() { _ = rows.Close() }()
	if err != nil {
		Fail(fmt.Sprintf("could not create function createCreds: %s", err.Error()))
	}
	if err := rows.Err(); err != nil {
		Fail(fmt.Sprintf(
			"error while createCreds: %s",
			err.Error(),
		))
	}
}

func addDeleteFunction() {
	rows, err := dbHelper.Query(fmt.Sprintf(`
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
	defer func() { _ = rows.Close() }()
	if err != nil {
		Fail(fmt.Sprintf("could not create function deleteCreds: %s", err.Error()))
	}
	if err := rows.Err(); err != nil {
		Fail(fmt.Sprintf(
			"error while deleteCreds: %s",
			err.Error(),
		))
	}
}

func addDeleteAllFunction() {
	rows, err := dbHelper.Query(fmt.Sprintf(`
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
	defer func() { _ = rows.Close() }()
	if err != nil {
		Fail(fmt.Sprintf("could not create function deleteAll: %s", err.Error()))
	}
	if err := rows.Err(); err != nil {
		Fail(fmt.Sprintf(
			"error while deleteAll: %s",
			err.Error(),
		))
	}
}

func addValidateFunction() {
	rows, err := dbHelper.Query(fmt.Sprintf(`
create or replace function "validate"( username text, password text)
returns TABLE(instance_id text, binding_id text)
language plpgsql
as $$
begin
	if username != '%s' or password != '%s' then
		 RAISE EXCEPTION 'invalid username and password' ;
	end if;
	return query SELECT username || ' from validate', password || ' from validate';
end;
$$`, instanceId, bindingId))
	defer func() { _ = rows.Close() }()
	if err != nil {
		Fail(fmt.Sprintf("could not create function validate: %s", err.Error()))
	}
	if err := rows.Err(); err != nil {
		Fail(fmt.Sprintf(
			"error while validate function: %s",
			err.Error(),
		))
	}
}

func deleteFunction(name string) {
	identifier := pgx.Identifier{"public", name}
	rows, err := dbHelper.Query(fmt.Sprintf("Drop function if exists %s", identifier.Sanitize()))
	defer func() { _ = rows.Close() }()
	if err != nil {
		Fail(fmt.Sprintf("could not remove procedure %s: %s", name, err.Error()))
	}
	if err := rows.Err(); err != nil {
		Fail(fmt.Sprintf(
			"error while remove procedure: %s",
			err.Error(),
		))
	}
}
