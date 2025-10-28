import React, { useEffect, useState } from "react";
import axios from "axios";
import "../styles/AdminPanel.css";

function AdminPanel({ token }) {
  const asItemsArray = (data) => {
    if (Array.isArray(data)) return data;
    if (data && Array.isArray(data.items)) return data.items;
    return [];
  };
  const asArray = (data) => (Array.isArray(data) ? data : []);

  const [products, setProducts] = useState([]);
  const [categories, setCategories] = useState([]);
  const [editingProduct, setEditingProduct] = useState(null);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [newCategory, setNewCategory] = useState("");

  const [formData, setFormData] = useState({
    name: "",
    description: "",
    price: "",
    categoryID: "",
    available: false,
    buyUrl: "",
  });
  const [file, setFile] = useState(null);

  useEffect(() => {
    fetchProducts();
    fetchCategories();
  }, []);

  const fetchProducts = async () => {
    try {
      const res = await axios.get("http://localhost:8080/api/read", {
        params: { page: 1, page_size: 1000 },
      });
      setProducts(asItemsArray(res.data));
    } catch {
      alert("Ошибка загрузки товаров");
      setProducts([]);
    }
  };

  const fetchCategories = async () => {
    try {
      const res = await axios.get("http://localhost:8080/api/categories");
      setCategories(asArray(res.data));
    } catch {
      alert("Ошибка загрузки категорий");
      setCategories([]);
    }
  };

  const handleCreateClick = () => {
    setFormData({
      name: "",
      description: "",
      price: "",
      categoryID: "",
      available: false,
      buyUrl: "",
    });
    setFile(null);
    setIsCreateModalOpen(true);
  };

  const handleInputChange = (e) => {
    const { name, value, type, checked } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: type === "checkbox" ? checked : value,
    }));
  };

  const handleFileChange = (e) => setFile(e.target.files[0] || null);

  const handleCreateCategory = async () => {
    if (!newCategory.trim()) return alert("Введите название категории");
    try {
      await axios.post(
        "http://localhost:8080/api/admin/category/create",
        { name: newCategory },
        { headers: { "X-Admin-Token": token } }
      );
      setNewCategory("");
      fetchCategories();
    } catch {
      alert("Ошибка при создании категории");
    }
  };

  const handleDeleteCategory = async (id) => {
    if (!window.confirm("Удалить категорию?")) return;
    try {
      await axios.post(
        "http://localhost:8080/api/admin/category/delete",
        { id },
        { headers: { "X-Admin-Token": token } }
      );
      fetchCategories();
    } catch {
      alert("Ошибка при удалении категории");
    }
  };

  const handleCreateSubmit = async (e) => {
    e.preventDefault();
    try {
      const data = new FormData();
      data.append("name", formData.name);
      data.append("description", formData.description);
      data.append("price", formData.price);
      data.append("categoryID", formData.categoryID);
      data.append("available", String(formData.available));
      data.append("buy_url", formData.buyUrl);
      if (file) data.append("file", file);

      await axios.post("http://localhost:8080/api/admin/create", data, {
        headers: {
          "X-Admin-Token": token,
          "Content-Type": "multipart/form-data",
        },
      });

      setIsCreateModalOpen(false);
      fetchProducts();
    } catch {
      alert("Ошибка при создании товара");
    }
  };

  const handleEditSubmit = async (e) => {
    e.preventDefault();
    try {
      const data = new FormData();
      data.append("id", editingProduct.id);
      data.append("name", formData.name);
      data.append("description", formData.description);
      data.append("price", formData.price);
      data.append("categoryID", formData.categoryID);
      data.append("available", String(formData.available));
      data.append("buy_url", formData.buyUrl);
      if (file) data.append("file", file);

      await axios.post("http://localhost:8080/api/admin/update", data, {
        headers: {
          "X-Admin-Token": token,
          "Content-Type": "multipart/form-data",
        },
      });

      setEditingProduct(null);
      fetchProducts();
    } catch {
      alert("Ошибка при обновлении товара");
    }
  };

  const handleDeleteClick = async (id) => {
    if (!window.confirm("Удалить товар?")) return;
    try {
      await axios.post(
        "http://localhost:8080/api/admin/delete",
        { id },
        { headers: { "X-Admin-Token": token } }
      );
      setProducts((prev) => prev.filter((p) => p.id !== id));
    } catch {
      alert("Ошибка при удалении товара");
    }
  };

  const openEditModal = (product) => {
    setEditingProduct(product);
    setFormData({
      name: product.name ?? "",
      description: product.description ?? "",
      price: String(product.price ?? ""),
      categoryID: String(product.categoryID ?? ""),
      available: Boolean(product.available),
      buyUrl: product.buyUrl ?? "",
    });
    setFile(null);
  };

  const closeModal = () => {
    setIsCreateModalOpen(false);
    setEditingProduct(null);
  };

  const modalTitle = isCreateModalOpen ? "Добавить новый товар" : "Редактирование товара";

  return (
    <div className="admin-panel">
      <h1>Админ-панель</h1>

      <div className="categories-section">
        <h2>Категории</h2>
        <div className="categories-list">
          {(categories || []).map((c) => (
            <div key={c.id} className="category-item">
              <span>{c.id}. {c.name}</span>
              <button className="delete-cat-btn" onClick={() => handleDeleteCategory(c.id)}>×</button>
            </div>
          ))}
        </div>

        <div className="category-create">
          <input
            type="text"
            value={newCategory}
            onChange={(e) => setNewCategory(e.target.value)}
            placeholder="Новая категория"
          />
          <button className="add-button" onClick={handleCreateCategory}>Создать категорию</button>
        </div>
      </div>

      <button onClick={handleCreateClick} className="add-button">
        Добавить новый товар
      </button>

      <table className="products-table">
        <thead>
          <tr>
            <th>ID</th>
            <th>Название</th>
            <th>Цена</th>
            <th>В наличии</th>
            <th>Категория</th>
            <th>Купить URL</th>
            <th>Картинка</th>
            <th>Действия</th>
          </tr>
        </thead>
        <tbody>
          {(products || []).map((p) => (
            <tr key={p.id}>
              <td>{p.id}</td>
              <td>{p.name}</td>
              <td>{p.price} ₽</td>
              <td>{p.available ? "Да" : "Нет"}</td>
              <td>{(categories || []).find((c) => Number(c.id) === Number(p.categoryID))?.name || "-"}</td>
              <td>{p.buyUrl ? <a href={p.buyUrl} target="_blank" rel="noreferrer">🔗</a> : "-"}</td>
              <td>
                <img
                  src={p.url || `http://localhost:8080/uploads/${p.id}.jpg`}
                  alt="preview"
                  style={{ width: 60, height: 60, objectFit: "cover" }}
                  onError={(e) => (e.currentTarget.style.display = "none")}
                />
              </td>
              <td>
                <button onClick={() => openEditModal(p)} className="edit-button">Редактировать</button>
                <button onClick={() => handleDeleteClick(p.id)} className="delete-button">Удалить</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {(isCreateModalOpen || editingProduct) && (
        <div className="modal">
          <div className="modal-content">
            <h3>{modalTitle}</h3>
            <form onSubmit={isCreateModalOpen ? handleCreateSubmit : handleEditSubmit}>
              <div className="form-group">
                <label>Название:</label>
                <input type="text" name="name" value={formData.name} onChange={handleInputChange} required />
              </div>

              <div className="form-group">
                <label>Описание:</label>
                <textarea name="description" value={formData.description} onChange={handleInputChange} rows={3} />
              </div>

              <div className="form-group">
                <label>Цена:</label>
                <input type="number" name="price" value={formData.price} onChange={handleInputChange} required />
              </div>

              <div className="form-group">
                <label>Ссылка на покупку (Buy URL):</label>
                <input
                  type="text"
                  name="buyUrl"
                  value={formData.buyUrl}
                  onChange={handleInputChange}
                  placeholder="https://example.com/item"
                />
              </div>

              <div className="form-group">
                <label>Категория:</label>
                <select name="categoryID" value={formData.categoryID} onChange={handleInputChange} required>
                  <option value="">-- выберите категорию --</option>
                  {(categories || []).map((c) => (
                    <option key={c.id} value={String(c.id)}>
                      {c.id}. {c.name}
                    </option>
                  ))}
                </select>
              </div>

              <div className="form-group checkbox">
                <label>
                  <input type="checkbox" name="available" checked={formData.available} onChange={handleInputChange} />
                  В наличии
                </label>
              </div>

              <div className="form-group">
                <label>Изображение:</label>
                <input type="file" accept="image/*" onChange={handleFileChange} />
                {editingProduct && (
                  <small>
                    Текущее изображение:{" "}
                    <a
                      href={editingProduct.url || `http://localhost:8080/uploads/${editingProduct.id}.jpg`}
                      target="_blank"
                      rel="noreferrer"
                    >
                      посмотреть
                    </a>
                  </small>
                )}
              </div>

              <div className="form-actions">
                <button type="submit">{isCreateModalOpen ? "Создать" : "Сохранить"}</button>
                <button type="button" onClick={closeModal}>Отмена</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}

export default AdminPanel;
