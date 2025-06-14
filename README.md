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
   http://localhost:5173
   ```
5. Use Swagger to test the API:
   ``` bash
   http://localhost:80/api/v1/swagger
   ```
6. Stop the Docker containers:
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

