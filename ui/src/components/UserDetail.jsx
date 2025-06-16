import React, { useState } from 'react';

const API_GATEWAY_URL = 'http://localhost:80';

function UserDetail() {
  const [userID, setUserID] = useState('');
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const handleFetchUser = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setUser(null);

    if (!userID) {
      setError('Введите ID пользователя.');
      setLoading(false);
      return;
    }

    try {
      const response = await fetch(`${API_GATEWAY_URL}/users/${userID}`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      setUser(data);
    } catch (e) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <h2>Информация о Пользователе</h2>
      <form onSubmit={handleFetchUser}>
        <label htmlFor="userIDInput">ID Пользователя:</label>
        <input
          type="text"
          id="userIDInput"
          value={userID}
          onChange={(e) => setUserID(e.target.value)}
          placeholder="Введите ID пользователя"
        />
        <button type="submit">Получить Информацию</button>
      </form>

      {loading && <p>Загрузка информации о пользователе...</p>}
      {error && <p className="error">Ошибка: {error}</p>}

      {user && (
        <div>
          <h3>Данные Пользователя:</h3>
          <p><strong>ID:</strong> {user.user_id}</p>
          <p><strong>Баланс:</strong> {user.balance || 'Не указано'}</p>
        </div>
      )}
    </div>
  );
}

export default UserDetail;