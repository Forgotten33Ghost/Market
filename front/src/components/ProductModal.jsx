import React, { useEffect, useCallback } from "react";
import "../styles/Modal.css";

export default function ProductModal({ isOpen, onClose, product = {}, onBuy }) {
  const {
    name = "",
    price = "",
    description = "",
    category = "",
    available = false,
    url = "",
    buyUrl = "",
  } = product;

  const handleKeyDown = useCallback(
    (e) => {
      if (e.key === "Escape") onClose?.();
    },
    [onClose]
  );

  useEffect(() => {
    if (!isOpen) return;
    document.body.style.overflow = "hidden";
    window.addEventListener("keydown", handleKeyDown);
    return () => {
      document.body.style.overflow = "";
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [isOpen, handleKeyDown]);

  if (!isOpen) return null;

  const handleBuy = () => {
    // если передан внешний обработчик — используем его
    if (typeof onBuy === "function") {
      onBuy();
      return;
    }
    // иначе открываем buyUrl
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

  const imageSrc = url || `images/${name}.webp`;

  return (
    <div
      className="modal-overlay"
      onClick={(e) => {
        if (e.target.classList.contains("modal-overlay")) onClose?.();
      }}
      role="dialog"
      aria-modal="true"
      aria-label={name ? `Карточка товара ${name}` : "Карточка товара"}
    >
      <div className="modal-card">
        <div className="modal-media">
          <img
            src={imageSrc}
            alt={name || "Фото товара"}
            onError={(e) => (e.currentTarget.src = "images/default.webp")}
          />
          {!available && <span className="modal-badge">Нет в наличии</span>}
        </div>

        <div className="modal-content">
          <h2 className="modal-title">{name || "Без названия"}</h2>

          {category && (
            <div className="modal-meta">
              Категория: <strong>{category}</strong>
            </div>
          )}

          {description && <p className="modal-desc">{description}</p>}

          {price !== "" && <div className="modal-price">{price}₽</div>}

          <div className="modal-actions">
            <button
              className="buy-btn"
              onClick={handleBuy}
              disabled={!available || !buyUrl}
              aria-disabled={!available || !buyUrl}
              title={!buyUrl ? "Ссылка на покупку отсутствует" : undefined}
            >
              Купить
            </button>
            <button className="ghost-btn" onClick={onClose}>
              Закрыть
            </button>
          </div>

          {/* Подсказка, если ссылки нет */}
          {!buyUrl && (
            <div className="modal-hint">
              <small>Добавьте ссылку «Купить» в админке для этого товара.</small>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
