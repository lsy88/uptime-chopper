import React, { useState, useRef, useEffect } from 'react';
import { FaMoon, FaSun, FaGlobe, FaSearch } from 'react-icons/fa';
import { Button, Form } from 'react-bootstrap';
import { useTranslation } from 'react-i18next';
import { useTheme } from '../contexts/ThemeContext';

interface TopBarProps {
  searchTerm: string;
  onSearch: (term: string) => void;
}

const TopBar: React.FC<TopBarProps> = ({ searchTerm, onSearch }) => {
  const { t, i18n } = useTranslation();
  const { theme, toggleTheme } = useTheme();
  const [isExpanded, setIsExpanded] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (isExpanded && inputRef.current) {
      inputRef.current.focus();
    }
  }, [isExpanded]);

  const handleBlur = () => {
    if (!searchTerm) {
      setIsExpanded(false);
    }
  };

  return (
    <div className="d-flex align-items-center gap-3">
      {/* Search */}
      <div className="d-flex align-items-center" style={{minHeight: '31px'}}>
        {isExpanded || searchTerm ? (
           <Form.Control
             ref={inputRef}
             size="sm"
             type="text"
             placeholder={t('search')}
             value={searchTerm}
             onChange={(e) => onSearch(e.target.value)}
             onBlur={handleBlur}
             className="bg-card border-secondary text-secondary"
             style={{width: '200px'}}
           />
        ) : (
          <Button 
            variant="link" 
            className="text-decoration-none text-secondary p-0 d-flex align-items-center"
            onClick={() => setIsExpanded(true)}
            title={t('search')}
          >
            <FaSearch size={18} />
          </Button>
        )}
      </div>

      <Button 
        variant="link" 
        className="text-decoration-none text-secondary p-0 d-flex align-items-center" 
        onClick={toggleTheme} 
        title={t('theme.' + theme)}
      >
        {theme === 'dark' ? <FaMoon size={20} /> : <FaSun size={20} />}
      </Button>
      <div className="d-flex align-items-center gap-2">
        <FaGlobe className="text-secondary" />
        <Form.Select 
          size="sm"
          className="bg-card text-secondary border-secondary" 
          style={{width: 'auto', cursor: 'pointer'}}
          value={i18n.language} 
          onChange={(e) => i18n.changeLanguage(e.target.value)}
        >
          <option value="en">English</option>
          <option value="zh">中文</option>
        </Form.Select>
      </div>
    </div>
  );
};

export default TopBar;
