FROM postgres:latest
ARG APP_PASSWORD
ARG APP_USER
ARG APP_DB

ENV APP_PASSWORD ${APP_PASSWORD}
ENV APP_USER ${APP_USER}
ENV APP_DB ${APP_DB}
ENV POSTGRES_HOST_AUTH_METHOD=password
ENV POSTGRES_PASSWORD=admin
ENV POSTGRES_USER=admin

VOLUME /tmp
COPY hospital_booking.sql /tmp/hospital_booking.sql

COPY init.sh /docker-entrypoint-initdb.d/init.sh
RUN sed -i 's/\r$//g' /docker-entrypoint-initdb.d/init.sh
RUN chmod 777 /docker-entrypoint-initdb.d/init.sh

EXPOSE 5432

