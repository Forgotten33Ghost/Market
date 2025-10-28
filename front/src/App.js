// src/App.js
import React, { useEffect, useState } from "react";
import axios from "axios";
import { BrowserRouter as Router, Routes, Route, useSearchParams } from "react-router-dom";
import AdminPanel from "./components/AdminPanel";
import Item from "./components/Item";
import SearchBar from "./components/SearchBar";
import SidebarFilters from "./components/SidebarFilters";

import "./styles/App.css";
import "./styles/AdminLogin.css";
import "./styles/Layout.css"; // макет: поиск сверху + фильтры слева

// небольшой хук-дебаунсер для поиска
function useDebounced(value, delay = 400) {
  const [v, setV] = useState(value);
  useEffect(() => {
    const t = setTimeout(() => setV(value), delay);
    return () => clearTimeout(t);
  }, [value, delay]);
  return v;
}

function ShopInner() {
  const [products, setProducts] = useState([]);
  const [categories, setCategories] = useState([]);
  const [searchParams, setSearchParams] = useSearchParams();

  // состояние фильтров и пагинации берём из URL (если есть)
  const [filters, setFilters] = useState({
    search: searchParams.get("search") ?? "",
    categoryId: searchParams.get("category_id") ?? "",
    minPrice: searchParams.get("min_price") ?? "",
    maxPrice: searchParams.get("max_price") ?? "",
    inStock: (searchParams.get("in_stock") ?? "") === "true",
    sort: searchParams.get("sort") ?? "price_asc",
    page: Number(searchParams.get("page") ?? 1),
    pageSize: Number(searchParams.get("page_size") ?? 24),
  });

  const debouncedSearch = useDebounced(filters.search, 400);

  // подгружаем категории один раз (ожидаем массив объектов {id, name})
  useEffect(() => {
    axios
      .get("http://localhost:8080/api/categories")
      .then((r) => setCategories(Array.isArray(r.data) ? r.data : []))
      .catch(() => setCategories([]));
  }, []);

  // синхронизация URL ←→ filters (без лишних дефолтов в адресной строке)
  useEffect(() => {
    const sp = new URLSearchParams();
    if (debouncedSearch) sp.set("search", debouncedSearch);
    if (filters.categoryId) sp.set("category_id", String(filters.categoryId));
    if (filters.minPrice) sp.set("min_price", String(filters.minPrice));
    if (filters.maxPrice) sp.set("max_price", String(filters.maxPrice));
    if (filters.inStock) sp.set("in_stock", "true");
    if (filters.sort !== "price_asc") sp.set("sort", filters.sort);
    if (filters.page !== 1) sp.set("page", String(filters.page));
    if (filters.pageSize !== 24) sp.set("page_size", String(filters.pageSize));

    // если пусто — чистим query, чтобы URL был /
    if ([...sp.keys()].length > 0) {
      setSearchParams(sp, { replace: true });
    } else {
      setSearchParams({}, { replace: true });
    }
  }, [debouncedSearch, filters, setSearchParams]);

  // по изменению query — тянем товары с сервера
  useEffect(() => {
    const controller = new AbortController();
    const params = Object.fromEntries(searchParams.entries());
    axios
      .get("http://localhost:8080/api/read", { params, signal: controller.signal })
      .then((r) => setProducts(Array.isArray(r.data) ? r.data : r.data.items || []))
      .catch((e) => {
        if (e.name !== "CanceledError") console.error("Ошибка загрузки товаров:", e);
        setProducts([]);
      });
    return () => controller.abort();
  }, [searchParams]);

  const resetFilters = () =>
    setFilters({
      search: "",
      categoryId: "",
      minPrice: "",
      maxPrice: "",
      inStock: false,
      sort: "price_asc",
      page: 1,
      pageSize: 24,
    });

  return (
    <div>
      <header>
        <h1>Добро пожаловать в магазин</h1>
      </header>

      {/* поиск сверху на всю ширину */}
      <SearchBar
        value={filters.search}
        onChange={(val) => setFilters((f) => ({ ...f, search: val, page: 1 }))}
      />

      {/* основной макет: слева фильтры, справа — товары */}
      <div className="main-grid">
        <SidebarFilters
          categories={categories}
          filters={filters}
          setFilters={(updater) => {
            setFilters((prev) => {
              const next = typeof updater === "function" ? updater(prev) : updater;
              // на любое изменение критериев — сбрасываем на 1-ю страницу
              if (next.page === prev.page) next.page = 1;
              return next;
            });
          }}
          onReset={resetFilters}
        />

        <div className="items-wrap">
          <div className="Items">
            {products.map((product) => (
              <Item
                key={product.id}
                name={product.name}
                price={product.price}
                description={product.description}
                category={product.category}
                available={product.available}
                url={product.url}           // картинка (первое изображение)
                buyUrl={product.buyUrl}     // ссылка «Купить»
              />
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

function Shop() {
  return <ShopInner />;
}

function AdminLogin() {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [token, setToken] = useState(null);

  const handleLogin = (e) => {
    e.preventDefault();
    axios
      .post("http://localhost:8080/api/admin/login", {
        login: username,
        password: password,
      })
      .then((response) => {
        setToken(response.data.token);
        alert("Успешный вход!");
      })
      .catch(() => alert("Ошибка авторизации"));
  };

  if (token) return <AdminPanel token={token} />;

  return (
    <div className="login">
      <h2>Добро пожаловать в админ-панель!</h2>
      <form onSubmit={handleLogin}>
        <div>
          <label>Логин:</label>
          <br />
          <input type="text" value={username} onChange={(e) => setUsername(e.target.value)} />
        </div>
        <div>
          <label>Пароль:</label>
          <br />
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
        </div>
        <button type="submit">Войти</button>
      </form>
    </div>
  );
}

export default function App() {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<Shop />} />
        <Route path="/admin" element={<AdminLogin />} />
      </Routes>
    </Router>
  );
}
