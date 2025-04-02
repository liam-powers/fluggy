
# script to open a psql connection with railway. don't forget to give password and port.
PGPASSWORD=$1 psql -h ballast.proxy.rlwy.net -U postgres -p $2 -d railway