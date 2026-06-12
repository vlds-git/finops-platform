import axios from 'axios';

const getToken = () => {
  if (typeof window !== 'undefined') {
    return localStorage.getItem('token');
  }
  return null;
};

const getCurrency = () => {
  if (typeof window !== 'undefined') {
    return localStorage.getItem('finops-currency') || 'USD';
  }
  return 'USD';
};

const redirectToLogin = () => {
  if (typeof window !== 'undefined') {
    window.location.href = '/login';
  }
};

const api = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL || '/api/v1',
  headers: {
    'Content-Type': 'application/json',
  },
});

api.interceptors.request.use((config) => {
  const token = getToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }

  // Inject currency preference into query params
  if (config.method?.toLowerCase() === 'get' && config.params !== false) {
    config.params = {
      ...config.params,
      currency: getCurrency(),
    };
  }

  return config;
});

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      if (typeof window !== 'undefined') {
        localStorage.removeItem('token');
      }
      redirectToLogin();
    }
    return Promise.reject(error);
  }
);

export default api;
