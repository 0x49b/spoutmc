import React, { useState } from 'react';
import { useNavigate, Navigate } from 'react-router-dom';
import { LoginPage as PFLoginPage, LoginForm } from '@patternfly/react-core';
import { useAuthStore } from '../../store/authStore';
import favicon from '../../assets/favicon.png';

const LoginPage: React.FC = () => {
  const navigate = useNavigate();
  const { login, user, loading } = useAuthStore();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');

  if (user) {
    return <Navigate to="/" replace />;
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      await login(email, password);
      navigate('/');
    } catch (error) {
      setError('Invalid email or password');
    }
  };

  return (
    <PFLoginPage
      brandImgSrc={favicon}
      brandImgAlt="SpoutMC"
      backgroundImgSrc="data:image/svg+xml,%3csvg xmlns='http://www.w3.org/2000/svg' width='4' height='4'%3e%3cpath fill='%239C92AC' fill-opacity='0.1' d='M1 3h1v1H1V3zm2-2h1v1H3V1z'%3e%3c/path%3e%3c/svg%3e"
      textContent="SpoutMC Server Management"
      loginTitle="Sign in to SpoutMC"
    >
      <LoginForm
        showHelperText={!!error}
        helperText={error}
        usernameLabel="Email"
        usernameValue={email}
        onChangeUsername={(_event, value) => setEmail(value)}
        passwordLabel="Password"
        passwordValue={password}
        onChangePassword={(_event, value) => setPassword(value)}
        onLoginButtonClick={handleSubmit}
        isLoginButtonDisabled={loading || !email || !password}
        loginButtonLabel={loading ? 'Signing in...' : 'Sign in'}
      />
    </PFLoginPage>
  );
};

export default LoginPage;
