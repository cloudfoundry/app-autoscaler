#!/usr/bin/env bash
sudo -S -u postgres psql -v ON_ERROR_STOP=true -c "CREATE EXTENSION pgcrypto;" autoscaler postgres
sudo -S -u postgres psql -v ON_ERROR_STOP=true -f install_sbss.sql autoscaler postgres
