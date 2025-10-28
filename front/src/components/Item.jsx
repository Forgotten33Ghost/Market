import React, { useState } from "react";
import "../styles/Item.css";
import ProductModal from "./ProductModal";

const Item = ({ name, price, description, category, available, url, buyUrl }) => {
  const [open, setOpen] = useState(false);

  const buy = () => {
    if (!available) {
      alert("Товар отсутствует в наличии");
      return;
    }
    if (buyUrl) {
      window.open(buyUrl, "_blank");
    } else {
      alert("Ссылка на покупку отсутствует");
    }
  };

  // то, что ждёт модалка в prop product
  const product = { name, price, description, category, available, url, buyUrl };

  return (
    <>
      <div className={`item-card ${available ? "" : "not-available"}`}>
        <img
          src={url ? url : `images/${name}.webp`}
          alt={name}
          onError={(e) => (e.currentTarget.src = "images/default.webp")}
        />
        <h1>{name}</h1>
        <p className="item-category"><strong>Категория:</strong> {category}</p>
        <div className="price">{price}₽</div>

        <div className="item-actions" style={{ display: "flex", gap: 12, width: "100%", justifyContent: "center" }}>
          <button onClick={() => setOpen(true)}>Подробнее</button>
          <button onClick={buy} disabled={!available}>Купить</button>
        </div>
      </div>

      <ProductModal
        isOpen={open}
        onClose={() => setOpen(false)}
        product={product}
        onBuy={buy}
      />
    </>
  );
};

export default Item;
