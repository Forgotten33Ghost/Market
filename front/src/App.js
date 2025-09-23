import React, { useEffect, useState } from "react";
import axios from 'axios';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import AdminPanel from './components/AdminPanel';
import Item from './components/Item'
import './styles/App.css'

function Shop() {
  const [products, setProducts] = useState([])
  useEffect(() => {
    axios.get('http://localhost:8080/api/read')  
      .then(response => {
        setProducts(Object.values(response.data))
      })
      .catch(error => console.error('Ошибка:', error));  
  }, []);

  return (
    <div>
      <header>
        <h1>Сюда можно что-то добавить</h1>
      </header>
      <div className="Items">
        {products.map((product) =>
          <Item
            key={product.id}
            name={product.name}
            price={product.price}
            gramm={product.gramm}
            available={product.available}
            url={product.url}
          />
        )}
      </div>
    </div>
  );
};

function AdminLogin() {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [isAuthenticated, setIsAuthenticated] = useState(false);

  const handleLogin = (e) => {
    e.preventDefault();

    axios.post('http://localhost:8080/api/admin/login', {"Login" : username, "Password" : password})
    .then(response => setIsAuthenticated(response))
    .catch(error => alert("Ошибка авторизации", error))
  };

  if (isAuthenticated) {
    return <AdminPanel />;
  }
    return (
      <div className="login">
        <h2>Добро пожаловать в админ-панель!</h2>
        <form onSubmit={handleLogin}>
          <div>
            <label>Логин:</label><br />
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
            />
          </div>
          <div>
            <label>Пароль:</label><br />
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>
          <button type="submit">Войти</button>
        </form>
      </div>
    );
}

function App() {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<Shop />} />
        <Route path="/admin" element={<AdminLogin />} />
      </Routes>
    </Router>
  );
}

export default App;
