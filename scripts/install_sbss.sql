create schema sys_xs_sbss;
set schema 'sys_xs_sbss';
\i '../sbss/sbss-postgresql/src/main/resources/db/migration/V1__SBSS.sql'
\i '../sbss/sbss-postgresql/src/main/resources/db/migration/V1_1__SBSS.sql'
\i '../sbss/sbss-postgresql/src/main/resources/db/migration/V1_2__SBSS.sql'
\i '../sbss/sbss-postgresql/src/main/resources/db/migration/V1_3__SBSS.sql'
drop user if exists sbss_test1;
create user sbss_test1 password 'test1234';
grant sbss_user to sbss_test1;

