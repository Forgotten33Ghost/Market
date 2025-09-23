import React, { useEffect, useState } from "react";
import axios from 'axios';

function AdminPanel() {
    const [products, setProducts] = useState([]);
    const [editingProduct, setEditingProduct] = useState(null);
    const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
    const [formData, setFormData] = useState({
        name: '',
        price: '',
        gramm: false,
        available: false,
        url: ''
    });

    useEffect(() => {
        fetchProducts();
    }, []);

    const fetchProducts = () => {
        axios.get('http://localhost:8080/api/read')
            .then(response => {
                setProducts(Object.values(response.data));
            })
            .catch(error => alert('Ошибка:', error));
    };

    const handleCreateClick = () => {
        setFormData({
            name: '',
            price: '',
            gramm: false,
            available: false,
            url: ''
        });
        setIsCreateModalOpen(true);
    };

    const handleCreateSubmit = (e) => {
    e.preventDefault();
    axios.post('http://localhost:8080/api/admin/create', {
        name: formData.name,
        price: Number(formData.price),
        gramm: formData.gramm,
        available: formData.available,
        i_URL: formData.i_URL
    })
        .then(() => {
            fetchProducts();
            setIsCreateModalOpen(false);
        })
        .catch(error => alert('Ошибка при создании:', error));
};


    const handleEditClick = (product) => {
        setEditingProduct(product);
        setFormData({
            name: product.name,
            price: product.price,
            gramm: product.gramm || false,
            available: product.available || false,
            url: product.url || ''
        });
    };

    const handleUpdateSubmit = (e) => {
        e.preventDefault();
        axios.put('http://localhost:8080/api/admin/update', {
            id: editingProduct.id,
            name: formData.name,
            price: Number(formData.price),
            gramm: formData.gramm,
            available: formData.available,
            url: formData.url
        })
        .then(() => {
            fetchProducts();
            setEditingProduct(null);
        })
        .catch(error => alert('Ошибка при обновлении:', error));
    };

    const handleDeleteClick = (id) => {
        if (window.confirm('Вы уверены, что хотите удалить этот товар?')) {
            axios.post('http://localhost:8080/api/admin/delete', { id })
                .then(() => {
                    fetchProducts();
                })
                .catch(error => alert('Ошибка при удалении:', error));
        }
    };

    const handleInputChange = (e) => {
        const { name, value, type, checked } = e.target;
        setFormData(prev => ({
            ...prev,
            [name]: type === 'checkbox' ? checked : value
        }));
    };

    const handleCancel = () => {
        setEditingProduct(null);
        setIsCreateModalOpen(false);
    };

    return (
        <div className="admin-panel">
            <h1>Админ-панель</h1>
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
                        <th>URL</th>
                        <th>Действия</th>
                    </tr>
                </thead>
                <tbody>
                    {products.map(product => (
                        <tr key={product.id}>
                            <td>{product.id}</td>
                            <td>{product.name}</td>
                            <td>{product.price} ₽{product.gramm ? '/100гр' : '/кг'}</td>
                            <td>{product.available ? 'Да' : 'Нет'}</td>
                            <td>
                              {product.url ? (
                                <a href={product.url} target="_blank" rel="noreferrer">{product.url}</a>
                              ) : (
                                '-'
                              )}
                            </td>
                            <td>
                                <button onClick={() => handleEditClick(product)} className="edit-button">
                                    Редактировать
                                </button>
                                <button onClick={() => handleDeleteClick(product.id)} className="delete-button">
                                    Удалить
                                </button>
                            </td>
                        </tr>
                    ))}
                </tbody>
            </table>

            {isCreateModalOpen && (
                <div className="modal">
                    <div className="modal-content">
                        <h3>Добавить новый товар</h3>
                        <form onSubmit={handleCreateSubmit}>
                            <div className="form-group">
                                <label>Название:</label>
                                <input
                                    type="text"
                                    name="name"
                                    value={formData.name}
                                    onChange={handleInputChange}
                                    required
                                />
                            </div>
                            <div className="form-group">
                                <label>Цена:</label>
                                <input
                                    type="number"
                                    name="price"
                                    value={formData.price}
                                    onChange={handleInputChange}
                                    required
                                />
                            </div>
                            <div className="form-group checkbox">
                                <label>
                                    <input
                                        type="checkbox"
                                        name="gramm"
                                        checked={formData.gramm}
                                        onChange={handleInputChange}
                                    />
                                    Цена за 100 грамм (иначе за кг)
                                </label>
                            </div>
                            <div className="form-group checkbox">
                                <label>
                                    <input
                                        type="checkbox"
                                        name="available"
                                        checked={formData.available}
                                        onChange={handleInputChange}
                                    />
                                    В наличии
                                </label>
                            </div>
                            <div className="form-group">
                                <label>URL:</label>
                                <input
                                    type="url"
                                    name="url"
                                    value={formData.url}
                                    onChange={handleInputChange}
                                    placeholder="https://example.com/image.jpg"
                                />
                            </div>
                            <div className="form-actions">
                                <button type="submit">Создать</button>
                                <button type="button" onClick={handleCancel}>Отмена</button>
                            </div>
                        </form>
                    </div>
                </div>
            )}

            {editingProduct && (
                <div className="modal">
                    <div className="modal-content">
                        <h3>Редактирование товара</h3>
                        <form onSubmit={handleUpdateSubmit}>
                            <div className="form-group">
                                <label>Название:</label>
                                <input
                                    type="text"
                                    name="name"
                                    value={formData.name}
                                    onChange={handleInputChange}
                                    required
                                />
                            </div>
                            <div className="form-group">
                                <label>Цена:</label>
                                <input
                                    type="number"
                                    name="price"
                                    value={formData.price}
                                    onChange={handleInputChange}
                                    required
                                />
                            </div>
                            <div className="form-group checkbox">
                                <label>
                                    <input
                                        type="checkbox"
                                        name="gramm"
                                        checked={formData.gramm}
                                        onChange={handleInputChange}
                                    />
                                    Цена за 100 грамм (иначе за кг)
                                </label>
                            </div>
                            <div className="form-group checkbox">
                                <label>
                                    <input
                                        type="checkbox"
                                        name="available"
                                        checked={formData.available}
                                        onChange={handleInputChange}
                                    />
                                    В наличии
                                </label>
                            </div>
                            <div className="form-group">
                                <label>URL:</label>
                                <input
                                    type="url"
                                    name="url"
                                    value={formData.url}
                                    onChange={handleInputChange}
                                    placeholder="https://example.com/image.jpg"
                                />
                            </div>
                            <div className="form-actions">
                                <button type="submit">Сохранить</button>
                                <button type="button" onClick={handleCancel}>Отмена</button>
                            </div>
                        </form>
                    </div>
                </div>
            )}
        </div>
    );
}

export default AdminPanel;
