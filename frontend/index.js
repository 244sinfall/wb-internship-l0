
document.getElementById('search-form').addEventListener('submit', (e) => {
    e.preventDefault()
    const orderId = document.getElementById('order-input').value
    const searchDb = document.getElementById('db-search').checked
    if(orderId) {
        window.location.href = `../orders/${orderId}`
    }
})

for(let item of document.querySelectorAll(".unix-time")) {
    item.innerHTML = new Date(Number(item.innerHTML * 1000)).toLocaleDateString("en-US")
}