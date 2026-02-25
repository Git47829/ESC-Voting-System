FROM mysql:8.2

ENV MYSQL_DATABASE=esc_voting
ENV MYSQL_USER=esc_user
ENV MYSQL_PASSWORD=esc_password
ENV MYSQL_ROOT_PASSWORD=secretroot

COPY db_scheme.sql /docker-entrypoint-initdb.d/01-schema.sql
COPY seed_data.sql /docker-entrypoint-initdb.d/02-seed.sql

EXPOSE 3306

CMD ["mysqld"]
