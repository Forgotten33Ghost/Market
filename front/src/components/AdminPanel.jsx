import React, { useEffect, useState } from "react";
import axios from 'axios';
import '../styles/AdminPanel.css';

function AdminPanel({ token }) {
    const [products, setProducts] = useState([]);
    const [categories, setCategories] = useState([]);
    const [editingProduct, setEditingProduct] = useState(null);
    const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
    const [formData, setFormData] = useState({
        name: '',
        description: '',
        price: '',
        categoryID: '',
        available: false,
        url: ''
    });

    useEffect(() => {
        fetchProducts();
        fetchCategories();
    }, []);

    const fetchProducts = () => {
        axios.get('http://localhost:8080/api/read')
            .then(response => setProducts(response.data))
            .catch(error => alert('Ошибка:', error));
    };

    const fetchCategories = () => {
        axios.get('http://localhost:8080/api/categories')
            .then(response => setCategories(response.data))
            .catch(error => alert('Ошибка получения категорий:', error));
    };

    const handleCreateClick = () => {
        setFormData({ name: '', description: '', price: '', categoryID: '', available: false, url: '' });
        setIsCreateModalOpen(true);
    };

    const handleCreateSubmit = (e) => {
        e.preventDefault();
        console.log("Данные:", formData.name, formData.description, Number(formData.price), Number(formData.category), formData.available, formData.url)
        axios.post('http://localhost:8080/api/admin/create', {
            name: formData.name,
            description: formData.description,
            price: Number(formData.price),
            categoryID: Number(formData.category),
            available: formData.available,
            url: formData.url
        }, {
            headers: { 'X-Admin-Token': token }
        })
        .then(() => {
            fetchProducts();
            setIsCreateModalOpen(false);
        })
        .catch(error => alert('Ошибка при создании:', error));
    };

    const handleInputChange = (e) => {
        const { name, value, type, checked } = e.target;
        setFormData(prev => ({ ...prev, [name]: type === 'checkbox' ? checked : value }));
    };

    const handleDeleteClick = (id) => {
        if (!window.confirm('Вы уверены, что хотите удалить этот товар?')) return;

        axios.post('http://localhost:8080/api/admin/delete', { id: id }, {
            headers: { 'X-Admin-Token': token }
        })
        .then(() => {
            setProducts(prev => prev.filter(p => p.id !== id));
        })
        .catch(error => {
            console.error('Ошибка при удалении товара:', error);
            alert('Ошибка при удалении товара');
        });
        };
    
    const modalTitle = isCreateModalOpen ? "Добавить новый товар" : "Редактирование товара";

    return (
        <div className="admin-panel">
            <h1>Админ-панель</h1>
            <button onClick={handleCreateClick} className="add-button">Добавить новый товар</button>

            {/* Таблица товаров */}
            <table className="products-table">
                <thead>
                    <tr>
                        <th>ID</th><th>Название</th><th>Цена</th><th>В наличии</th><th>Категория</th><th>Действия</th>
                    </tr>
                </thead>
                <tbody>
                    {products.map(product => (
                        <tr key={product.id}>
                            <td>{product.id}</td>
                            <td>{product.name}</td>
                            <td>{product.price} ₽</td>
                            <td>{product.available ? 'Да' : 'Нет'}</td>
                            <td>{categories.find(c => c.id === product.categoryID)?.name || '-'}</td>
                            <td>
                                <button onClick={() => setEditingProduct(product)} className="edit-button">Редактировать</button>
                                <button onClick={() => handleDeleteClick(product.id)} className="delete-button">Удалить</button>
                            </td>
                        </tr>
                    ))}
                </tbody>
            </table>

            {/* Модальное окно */}
            {(isCreateModalOpen || editingProduct) && (
            <div className="modal">
                <div className="modal-content">
                    <h3>{modalTitle}</h3>
                    <form onSubmit={isCreateModalOpen ? handleCreateSubmit : () => {}}>
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
                            <label>Категория:</label>
                            <select name="category" value={formData.category} onChange={handleInputChange} required>
                            <option value="">-- выберите категорию --</option>
                            {categories.map(c => (
                                <option key={c.id} value={c.id}>
                                    {c.id}.{c.name}
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
                            <label>URL:</label>
                            <input type="url" name="url" value={formData.url} onChange={handleInputChange} placeholder="https://example.com/" />
                        </div>

                        <div className="form-actions">
                            <button type="submit">{isCreateModalOpen ? "Создать" : "Сохранить"}</button>
                            <button type="button" onClick={() => { setEditingProduct(null); setIsCreateModalOpen(false); }}>Отмена</button>
                        </div>
                    </form>
                </div>
            </div>
            )}
        </div>
    );
}

export default AdminPanel;
