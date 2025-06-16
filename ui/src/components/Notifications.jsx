import React, { useEffect, useState } from 'react';

// TODO: Написать компонент Notifications, который будет подключаться к WebSocket-серверу и отображать уведомления о статусах заказов.

const WEBSOCKET_URL = 'ws://localhost:8082/ws/orders'; 

function Notifications() {
  const [messages, setMessages] = useState([]);
  const [connectionStatus, setConnectionStatus] = useState('Подключение...');

  useEffect(() => {
    const ws = new WebSocket(WEBSOCKET_URL);

    ws.onopen = () => {
      setConnectionStatus('Подключено');
      console.log('Connected to WebSocket server');
      // Запрос разрешения на push-уведомления при успешном подключении
      if (Notification.permission !== "granted" && Notification.permission !== "denied") {
        Notification.requestPermission();
      }
    };

    ws.onmessage = (event) => {
      try {
        const newMessage = JSON.parse(event.data);
        setMessages(prevMessages => [...prevMessages, newMessage]);
        console.log('Received WebSocket message:', newMessage);

        // Показать браузерное push-уведомление
        if (Notification.permission === "granted") {
          const notificationBody = `Заказ ${newMessage.orderId || 'N/A'} статус: ${newMessage.status || 'N/A'}`;
          new Notification("Обновление Заказа", { body: notificationBody });
        }
      } catch (e) {
        console.error('Failed to parse WebSocket message:', e, event.data);
      }
    };

    ws.onclose = () => {
      setConnectionStatus('Отключено');
      console.log('Disconnected from WebSocket server');
    };

    ws.onerror = (error) => {
      setConnectionStatus('Ошибка подключения');
      console.error('WebSocket error:', error);
    };

    // Очистка при размонтировании компонента: закрыть WebSocket-соединение
    return () => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.close();
      }
    };
  }, []); // Пустой массив зависимостей: эффект запускается один раз при монтировании

  return (
    <div>
      <h2>Уведомления о Заказах (WebSocket)</h2>
      <p>Статус соединения: <strong>{connectionStatus}</strong></p>
      <ul>
        {messages.map((msg, index) => (
          <li key={index}>
            <strong>Заказ ID:</strong> {msg.orderId || 'N/A'}, 
            <strong> Статус:</strong> {msg.status || 'N/A'} 
            (Время: {new Date(msg.timestamp || Date.now()).toLocaleTimeString()})
          </li>
        ))}
      </ul>
      {messages.length === 0 && connectionStatus === 'Подключено' && (
        <p>Ожидаем уведомлений о статусах заказов...</p>
      )}
    </div>
  );
}

export default Notifications;