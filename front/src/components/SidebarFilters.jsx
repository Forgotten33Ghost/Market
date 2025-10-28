// components/SidebarFilters.jsx
import React from "react";
import "../styles/Layout.css";

export default function SidebarFilters({ categories, filters, setFilters, onReset }) {
  return (
    <aside className="sidebar">
      <div className="sidebar-group">
        <label>Категория</label>
        <select
          value={filters.categoryId}
          onChange={(e) => setFilters(f => ({ ...f, categoryId: e.target.value }))}
        >
          <option value="">Все категории</option>
          {categories.map(c => (
            <option key={c.id} value={String(c.id)}>{c.name}</option>
          ))}
        </select>
      </div>

      <div className="sidebar-group">
        <label>Цена</label>
        <div className="price-row">
          <input
            type="number" placeholder="от" min="0"
            value={filters.minPrice}
            onChange={(e) => setFilters(f => ({ ...f, minPrice: e.target.value }))}
          />
          <span>—</span>
          <input
            type="number" placeholder="до" min="0"
            value={filters.maxPrice}
            onChange={(e) => setFilters(f => ({ ...f, maxPrice: e.target.value }))}
          />
        </div>
      </div>

      <label className="checkbox">
        <input
          type="checkbox"
          checked={filters.inStock}
          onChange={(e) => setFilters(f => ({ ...f, inStock: e.target.checked }))}
        />
        Только в наличии
      </label>

      <div className="sidebar-group">
        <label>Сортировка</label>
        <select
          value={filters.sort}
          onChange={(e) => setFilters(f => ({ ...f, sort: e.target.value }))}
        >
          <option value="price_asc">Цена: сначала дешевле</option>
          <option value="price_desc">Цена: сначала дороже</option>
          <option value="name_asc">Название: A–Z</option>
          <option value="name_desc">Название: Z–A</option>
        </select>
      </div>

      <button className="reset-btn" onClick={onReset}>Сбросить</button>
    </aside>
  );
}
