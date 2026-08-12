import React, { useState, useEffect } from 'react';

const API_BASE = window.location.origin.includes('-3000.')
  ? window.location.origin.replace('-3000.', '-8080.') // Codespaces環境用
  : `${window.location.protocol}//${window.location.hostname}:8080`; // AWS / ローカル環境用

function App() {
  const [tableNo, setTableNo] = useState(1);
  const [menuItems, setMenuItems] = useState([]);
  const [activeCategory, setActiveCategory] = useState('すべて');
  const [cart, setCart] = useState([]);
  const [orderHistory, setOrderHistory] = useState([]);
  const [activeTab, setActiveTab] = useState('menu'); // 'menu' | 'cart' | 'history'
  const [checkoutModalOpen, setCheckoutModalOpen] = useState(false);
  const [billSummary, setBillSummary] = useState(null);

  const fetchOptions = (options = {}) => {
    return {
      ...options,
      credentials: 'include',
    };
  };

  useEffect(() => {
    fetchMenu();
    fetchOrderHistory();
  }, [tableNo]);

  const fetchMenu = async () => {
    try {
      const res = await fetch(`${API_BASE}/api/menu`, fetchOptions());
      if (res.ok) {
        const data = await res.json();
        setMenuItems(data || []);
      }
    } catch (e) {
      console.error("メニュー取得エラー:", e);
    }
  };

  const fetchOrderHistory = async () => {
    try {
      const res = await fetch(`${API_BASE}/api/orders/history`, fetchOptions({
        headers: { 'X-Table-No': String(tableNo) }
      }));
      if (res.ok) {
        const data = await res.json();
        setOrderHistory(data || []);
      }
    } catch (e) {
      console.error("注文履歴取得エラー:", e);
    }
  };

  const addToCart = (item) => {
    setCart((prev) => {
      const existing = prev.find((i) => i.id === item.id);
      if (existing) {
        return prev.map((i) => i.id === item.id ? { ...i, quantity: i.quantity + 1 } : i);
      }
      return [...prev, { ...item, quantity: 1 }];
    });
  };

  const updateCartQuantity = (id, delta) => {
    setCart((prev) => {
      return prev.map((item) => {
        if (item.id === id) {
          const newQty = item.quantity + delta;
          return newQty > 0 ? { ...item, quantity: newQty } : null;
        }
        return item;
      }).filter(Boolean);
    });
  };

  const submitOrder = async () => {
    if (cart.length === 0) return;

    const payload = {
      table_no: tableNo,
      items: cart.map((i) => ({ menu_item_id: i.id, quantity: i.quantity }))
    };

    try {
      const res = await fetch(`${API_BASE}/api/orders`, fetchOptions({
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Table-No': String(tableNo)
        },
        body: JSON.stringify(payload)
      }));

      if (res.ok) {
        alert('ご注文を承りました！');
        setCart([]);
        fetchOrderHistory();
        setActiveTab('history');
      } else {
        alert('注文処理に失敗しました。');
      }
    } catch (e) {
      console.error(e);
      alert('通信エラーが発生しました。');
    }
  };

  const handleOpenCheckout = async () => {
    try {
      const res = await fetch(`${API_BASE}/api/checkout`, fetchOptions({
        headers: { 'X-Table-No': String(tableNo) }
      }));
      if (res.ok) {
        const data = await res.json();
        setBillSummary(data);
        setCheckoutModalOpen(true);
      }
    } catch (e) {
      console.error(e);
    }
  };

  const confirmCheckout = async () => {
    try {
      const res = await fetch(`${API_BASE}/api/checkout`, fetchOptions({
        method: 'POST',
        headers: { 'X-Table-No': String(tableNo) }
      }));
      if (res.ok) {
        alert('精算が完了しました。ご来店ありがとうございました！');
        setCheckoutModalOpen(false);
        setOrderHistory([]);
        setCart([]);
        setActiveTab('menu');
      }
    } catch (e) {
      console.error(e);
    }
  };

  const categories = ['すべて', ...new Set(menuItems.map(i => i.category))];

  const filteredMenuItems = activeCategory === 'すべて' 
    ? menuItems 
    : menuItems.filter(i => i.category === activeCategory);

  const cartTotalAmount = cart.reduce((sum, item) => sum + (item.price * item.quantity), 0);
  const cartTotalItems = cart.reduce((sum, item) => sum + item.quantity, 0);

  return (
    <div className="app-container">
      <header className="navbar">
        <div className="nav-content">
          <h1 className="logo">居酒屋 摩摩</h1>
          <div className="table-selector">
            <label htmlFor="table-select">卓番号: </label>
            <input 
              id="table-select"
              type="number" 
              min="1" 
              value={tableNo} 
              onChange={(e) => setTableNo(Number(e.target.value))}
            />
          </div>
        </div>
      </header>

      <nav className="tab-bar">
        <button className={activeTab === 'menu' ? 'active' : ''} onClick={() => setActiveTab('menu')}>
          メニュー選択
        </button>
        <button className={activeTab === 'cart' ? 'active' : ''} onClick={() => setActiveTab('cart')}>
          注文確認 {cartTotalItems > 0 && <span className="badge">{cartTotalItems}</span>}
        </button>
        <button className={activeTab === 'history' ? 'active' : ''} onClick={() => setActiveTab('history')}>
          注文履歴
        </button>
        <button className="btn-checkout-nav" onClick={handleOpenCheckout}>
          お会計・精算
        </button>
      </nav>

      <main className="main-layout">
        {activeTab === 'menu' && (
          <section className="menu-section">
            <div className="category-list">
              {categories.map((cat) => (
                <button 
                  key={cat} 
                  className={`category-chip ${activeCategory === cat ? 'active' : ''}`}
                  onClick={() => setActiveCategory(cat)}
                >
                  {cat}
                </button>
              ))}
            </div>

            <div className="menu-grid">
              {filteredMenuItems.map((item) => (
                <div key={item.id} className="menu-card card">
                  <img src={item.image_url} alt={item.name} className="menu-img" />
                  <div className="menu-info">
                    <span className="menu-category">{item.category}</span>
                    <h3 className="menu-title">{item.name}</h3>
                    <p className="menu-desc">{item.description}</p>
                    <div className="menu-bottom">
                      <span className="menu-price">¥{item.price.toLocaleString()}</span>
                      <button className="btn-primary" onClick={() => addToCart(item)}>
                        カートに追加
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </section>
        )}

        {activeTab === 'cart' && (
          <section className="cart-section card">
            <h2>現在の注文リスト</h2>
            {cart.length === 0 ? (
              <p className="empty-message">カートに商品がありません。</p>
            ) : (
              <>
                <div className="cart-list">
                  {cart.map((item) => (
                    <div key={item.id} className="cart-item">
                      <div className="cart-item-info">
                        <h4>{item.name}</h4>
                        <p>¥{item.price.toLocaleString()} × {item.quantity}</p>
                      </div>
                      <div className="qty-controls">
                        <button onClick={() => updateCartQuantity(item.id, -1)}>-</button>
                        <span>{item.quantity}</span>
                        <button onClick={() => updateCartQuantity(item.id, 1)}>+</button>
                      </div>
                    </div>
                  ))}
                </div>
                <div className="cart-summary">
                  <div className="total-row">
                    <span>小計 (税込):</span>
                    <span className="total-price">¥{cartTotalAmount.toLocaleString()}</span>
                  </div>
                  <button className="btn-order-submit" onClick={submitOrder}>
                    厨房へ注文を確定する ({cartTotalItems}点)
                  </button>
                </div>
              </>
            )}
          </section>
        )}

        {activeTab === 'history' && (
          <section className="history-section card">
            <h2>注文済み履歴（卓 No.{tableNo}）</h2>
            {orderHistory.length === 0 ? (
              <p className="empty-message">まだ注文履歴はありません。</p>
            ) : (
              <table className="history-table">
                <thead>
                  <tr>
                    <th>品名</th>
                    <th>単価</th>
                    <th>数量</th>
                    <th>小計</th>
                    <th>時間</th>
                  </tr>
                </thead>
                <tbody>
                  {orderHistory.map((item) => (
                    <tr key={item.id}>
                      <td>{item.item_name}</td>
                      <td>¥{item.price.toLocaleString()}</td>
                      <td>{item.quantity}</td>
                      <td>¥{item.subtotal.toLocaleString()}</td>
                      <td>{new Date(item.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </section>
        )}
      </main>

      {checkoutModalOpen && billSummary && (
        <div className="modal-overlay">
          <div className="modal-card card">
            <h2>お会計確認 (卓 No.{billSummary.table_no})</h2>
            <div className="checkout-details">
              {billSummary.details.length === 0 ? (
                <p>注文データがありません。</p>
              ) : (
                <ul className="checkout-list">
                  {billSummary.details.map((item) => (
                    <li key={item.id}>
                      <span>{item.item_name} × {item.quantity}</span>
                      <span>¥{item.subtotal.toLocaleString()}</span>
                    </li>
                  ))}
                </ul>
              )}
              <hr />
              <div className="total-row modal-total">
                <span>合計金額 ({billSummary.total_items}点):</span>
                <span className="total-price">¥{billSummary.total_amount.toLocaleString()}</span>
              </div>
            </div>
            <div className="modal-actions">
              <button className="btn-secondary" onClick={() => setCheckoutModalOpen(false)}>戻る</button>
              <button 
                className="btn-primary btn-confirm-checkout" 
                onClick={confirmCheckout}
                disabled={billSummary.total_amount === 0}
              >
                精算する (退店)
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default App;