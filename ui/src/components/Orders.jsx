import React, { useState, useEffect } from 'react';

const API_GATEWAY_URL = import.meta.env.REACT_APP_API_GATEWAY_URL || 'http://localhost:80';

function Orders() {
  const [orders, setOrders] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const [newOrderUserID, setNewOrderUserID] = useState('');
  const [newOrderAmount, setNewOrderAmount] = useState('');
  const [description, setDescription] = useState('');
  const [createOrderError, setCreateOrderError] = useState(null);
  const [createOrderSuccess, setCreateOrderSuccess] = useState(false);

  const fetchOrders = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(`${API_GATEWAY_URL}/orders`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      setOrders(data);
    } catch (e) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchOrders();
  }, []);


  const handleCreateOrder = async (e) => {
    e.preventDefault();
    setCreateOrderError(null);
    setCreateOrderSuccess(false);

    if (!newOrderUserID || !newOrderAmount || isNaN(newOrderAmount)) {
      setCreateOrderError('Все поля обязательны для заполнения.');
      return;
    }

    try {
      const response = await fetch(`${API_GATEWAY_URL}/orders`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          user_id: parseInt(newOrderUserID),
          amount: parseFloat(newOrderAmount), 
          description: description,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(`Ошибка создания заказа: ${response.status} - ${errorData.message || 'Неизвестная ошибка'}`);
      }

      const createdOrder = await response.json();
      setCreateOrderSuccess(true);
      setNewOrderUserID('');
      setNewOrderAmount('');
      setDescription('');
      fetchOrders();
      console.log('Заказ успешно создан:', createdOrder);

    } catch (e) {
      setCreateOrderError(e.message);
    }
  };


  return (
    <div>
      <h2>Список Заказов</h2>
      {loading && <p>Загрузка заказов...</p>}
      {error && <p className="error">Ошибка при загрузке заказов: {error}</p>}
      {!loading && !error && orders.length === 0 && <p>Заказов пока нет.</p>}
      <ul>
        {orders.map(order => (
          <li key={order.id}>
            <strong>ID:</strong> {order.id}, 
            <strong> Пользователь:</strong> {order.user_id}, 
            <strong> Сумма:</strong> {order.amount}, 
            <strong> Статус:</strong> {order.status}
          </li>
        ))}
      </ul>

      <h3>Создать Новый Заказ</h3>
      <form onSubmit={handleCreateOrder}>
        <div>
          <label htmlFor="user_id">ID Пользователя:</label>
          <input
            type="text"
            id="user_id"
            value={newOrderUserID}
            onChange={(e) => setNewOrderUserID(e.target.value)}
            placeholder="Введите ID пользователя"
          />
        </div>
        <div>
          <label htmlFor="amount">Сумма:</label>
          <input
            type="number"
            id="amount"
            step="0.01"
            value={newOrderAmount}
            onChange={(e) => setNewOrderAmount(e.target.value)}
            placeholder="Введите сумму"
          />
        </div>
        <div>
          <label htmlFor="description">Описание:</label>
          <input
            type="text"
            id="description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Введите описание"
          />
        </div>
        <button type="submit">Создать Заказ</button>
        {createOrderError && <p className="error">{createOrderError}</p>}
        {createOrderSuccess && <p className="success">Заказ успешно создан!</p>}
      </form>
      <button onClick={fetchOrders}>Обновить Список Заказов</button>
    </div>
  );
}

export default Orders;