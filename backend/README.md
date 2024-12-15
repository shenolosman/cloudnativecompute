# 4 different services 

product and order service is written with golang and auth and notification service is written wih pyhton.

## They communicate each other with RabbitMQ 

Purpose is here to create microservices att sends api endpoints to frontend for project.

Using docker-compose:

´´´bash
    cd backend
    docker-compose up -d
´´´

or you can individually put services up but needs manuel docker connection for required service(redis, rabbitmq or sql)