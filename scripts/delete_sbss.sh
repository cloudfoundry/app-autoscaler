#!/usr/bin/env bash

sudo -S -u postgres psql -v ON_ERROR_STOP=true -c "DROP SCHEMA sys_xs_sbss CASCADE;" autoscaler postgres
sudo -S -u postgres psql -v ON_ERROR_STOP=true -c "DROP USER sbss_test1;" autoscaler postgres
sudo -S -u postgres psql -v ON_ERROR_STOP=true -c "DROP EXTENSION pgcrypto;" autoscaler postgres
