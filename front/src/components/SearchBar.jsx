import React from "react";
import "../styles/Layout.css";

export default function SearchBar({ value, onChange }) {
  return (
    <div className="searchbar-wrap">
      <input
        type="text"
        className="searchbar-input"
        placeholder="Поиск по названию или описанию…"
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
    </div>
  );
}
