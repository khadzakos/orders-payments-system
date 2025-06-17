# orders-payments-system

Implementation of Payments Service and Orders Service — to ensure the stable operation of the online store under high loads.

### How to run
1. Clone the repository:
   ```bash
    git clone https://github.com/khadzakos/orders-payments-system.git
    cd orders-payments-system
    ```
2. Create .env file at the root of the project and add the following variables:
   ```bash
   ORDERS_DB_HOST=postgres_orders
   ORDERS_DB_USER=orderuser
   ORDERS_DB_PASSWORD=orderpass
   ORDERS_DB_NAME=ordersdb

   PAYMENTS_DB_HOST=postgres_payments
   PAYMENTS_DB_USER=paymentuser
   PAYMENTS_DB_PASSWORD=paymentpass
   PAYMENTS_DB_NAME=paymentsdb
   ```
3. Start the Docker containers:
   ```bash
   docker-compose up -d
   ```
4. Use UI to try the application:
   ``` bash
   http://localhost:3000
   ```
5. Use Postman Collection to test the API
7. Stop the Docker containers:
   ```bash
   docker-compose down
   ```
   To clear volumes:
   ```bash
   docker-compose down -v
   ```

### Technologies used
- **Golang**: Main programming language for the services.
- **PostgreSQL**: Database for storing orders and payments.
- **Docker**: Containerization for easy deployment and management.
- **Chi**: HTTP router for handling requests.
- **Swagger**: API documentation and testing.
- **Kafka**: Message broker for asynchronous communication between services.
- **Zap**: For advanced logging capabilities.
- **Vite + React**: Frontend for the application.

### Описание проекта

Данный проект представляет собой систему заказов и оплат, реализованную с использованием микросервисной архитектуры на основе паттернов Transactional Outbox и Transactional Inbox.

В папке `gateway` реализован API Gateway.
В папке `orders` реализован сервис заказов.
В папке `payments` реализован сервис оплат.
В папке `ui` реализован клиентский интерфейс.

Архитектура микросервисов `orders` и `payments` построена на основных понятиях чистой архитектуры. Для взаимодействия между сервисами используется Kafka.

У `orders` микросевиса есть таблица заказов - `orders` и outbox таблица - `outbox`. Outbox таблица используется для отправки сообщений из Kafka, которые представляют собой запросы на создание заказов.

У `payments` микросевиса есть таблица оплат - `payments`, inbox таблица - `inbox_messages` и outbox таблица - `outbox_messages`, а также таблица с транзакциями - `payments`. Outbox таблица используется для отправки результатов обработки сообщений из inbox таблицы, то есть содержит информацию о том, какие заказы были успешно обработаны, а какие - нет. Inbox таблица используется для получения сообщений из Kafka, которые представляют собой запросы на оплату.

В Kafka есть два топика: `order_payment_tasks` и `payment_status_updates`. Топик `order_payment_tasks` используется для отправки сообщений о создании заказа из outbox таблицы orders-service, а топик `payment_status_updates` используется для отправки сообщений о статусе заказа из outbox таблицы payments-service.

Фронтэнд реализован на React и представляет из себя простой интерфейс для создания заказов, просмотра списка заказов, а также баланса юзера.

В папке `postman` находится коллекция Postman для тестирования API. Тесты к приложению можно найти в соответвующих модулях в файлах с пометкой test.

TODO:
Планируется добавить push-уведомления через websockets(на ui есть соответствующее окно, однако, пока оно ничего не выводит).

Каждый сервис имеет свой Dockerfile, сборка и запуск осуществляется через Docker Compose.
