import { Role, UserProfile } from '../types';

// Mock users for development
const mockUsers = [
  {
    email: 'admin@example.com',
    password: 'password',
    roles: ['admin'],
    displayName: 'Admin User'
  },
  {
    email: 'mod@example.com',
    password: 'password',
    roles: ['moderator', 'viewer'],
    displayName: 'Moderator User'
  },
  {
    email: 'viewer@example.com',
    password: 'password',
    roles: ['viewer'],
    displayName: 'Viewer User'
  }
];

// Mock JWT secret (in a real app, this would be on the server)
const JWT_SECRET = 'your-jwt-secret';

// Mock JWT token generation
function generateToken(user: typeof mockUsers[0]): string {
  const payload = {
    sub: user.email,
    email: user.email,
    roles: user.roles,
    displayName: user.displayName,
    iat: Math.floor(Date.now() / 1000),
    exp: Math.floor(Date.now() / 1000) + (24 * 60 * 60) // 24 hours
  };
  
  // In a real app, this would be done on the server
  // For mock purposes, we'll just encode the payload
  return btoa(JSON.stringify(payload));
}

export async function mockLogin(email: string, password: string): Promise<{ token: string }> {
  const user = mockUsers.find(u => u.email === email && u.password === password);
  
  if (!user) {
    throw new Error('Invalid credentials');
  }
  
  return { token: generateToken(user) };
}

export function mockVerifyToken(token: string): UserProfile {
  try {
    const decoded = JSON.parse(atob(token));
    
    if (decoded.exp < Math.floor(Date.now() / 1000)) {
      throw new Error('Token expired');
    }
    
    return {
      id: decoded.sub,
      email: decoded.email,
      roles: decoded.roles,
      permissions: [],
      displayName: decoded.displayName,
      created_at: new Date().toISOString(),
      lastLoginAt: new Date().toISOString(),
      aud: 'authenticated',
      app_metadata: {},
      user_metadata: {},
      identities: []
    };
  } catch (error) {
    throw new Error('Invalid token');
  }
}