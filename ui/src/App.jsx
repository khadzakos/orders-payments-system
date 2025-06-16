import React from 'react';
import Orders from './components/Orders';
import UserDetail from './components/UserDetail';
import Notifications from './components/Notifications';

function App() {
  return (
    <div>
      <h1>Панель управления Магазином</h1>

      <Orders /> {/* Компонент для заказов */}
      <hr />
      <UserDetail /> {/* Компонент для информации о пользователе */}
      <hr />
      <Notifications /> {/* Компонент для уведомлений WebSocket */}
    </div>
  );
}

export default App;